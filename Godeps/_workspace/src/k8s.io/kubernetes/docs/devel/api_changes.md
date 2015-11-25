<!-- BEGIN MUNGE: UNVERSIONED_WARNING -->

<!-- BEGIN STRIP_FOR_RELEASE -->

<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">
<img src="http://kubernetes.io/img/warning.png" alt="WARNING"
     width="25" height="25">

<h2>PLEASE NOTE: This document applies to the HEAD of the source tree</h2>

If you are using a released version of Kubernetes, you should
refer to the docs that go with that version.

<strong>
The latest 1.0.x release of this document can be found
[here](http://releases.k8s.io/release-1.0/docs/devel/api_changes.md).

Documentation for other releases can be found at
[releases.k8s.io](http://releases.k8s.io).
</strong>
--

<!-- END STRIP_FOR_RELEASE -->

<!-- END MUNGE: UNVERSIONED_WARNING -->

# So you want to change the API?

Before attempting a change to the API, you should familiarize yourself
with a number of existing API types and with the [API
conventions](api-conventions.md).  If creating a new API
type/resource, we also recommend that you first send a PR containing
just a proposal for the new API types, and that you initially target
the experimental API (pkg/apis/experimental).

The Kubernetes API has two major components - the internal structures and
the versioned APIs.  The versioned APIs are intended to be stable, while the
internal structures are implemented to best reflect the needs of the Kubernetes
code itself.

What this means for API changes is that you have to be somewhat thoughtful in
how you approach changes, and that you have to touch a number of pieces to make
a complete change.  This document aims to guide you through the process, though
not all API changes will need all of these steps.

## Operational overview

It is important to have a high level understanding of the API system used in
Kubernetes in order to navigate the rest of this document.

As mentioned above, the internal representation of an API object is decoupled
from any one API version.  This provides a lot of freedom to evolve the code,
but it requires robust infrastructure to convert between representations.  There
are multiple steps in processing an API operation - even something as simple as
a GET involves a great deal of machinery.

The conversion process is logically a "star" with the internal form at the
center. Every versioned API can be converted to the internal form (and
vice-versa), but versioned APIs do not convert to other versioned APIs directly.
This sounds like a heavy process, but in reality we do not intend to keep more
than a small number of versions alive at once.  While all of the Kubernetes code
operates on the internal structures, they are always converted to a versioned
form before being written to storage (disk or etcd) or being sent over a wire.
Clients should consume and operate on the versioned APIs exclusively.

To demonstrate the general process, here is a (hypothetical) example:

   1. A user POSTs a `Pod` object to `/api/v7beta1/...`
   2. The JSON is unmarshalled into a `v7beta1.Pod` structure
   3. Default values are applied to the `v7beta1.Pod`
   4. The `v7beta1.Pod` is converted to an `api.Pod` structure
   5. The `api.Pod` is validated, and any errors are returned to the user
   6. The `api.Pod` is converted to a `v6.Pod` (because v6 is the latest stable
      version)
   7. The `v6.Pod` is marshalled into JSON and written to etcd

Now that we have the `Pod` object stored, a user can GET that object in any
supported api version.  For example:

   1. A user GETs the `Pod` from `/api/v5/...`
   2. The JSON is read from etcd and unmarshalled into a `v6.Pod` structure
   3. Default values are applied to the `v6.Pod`
   4. The `v6.Pod` is converted to an `api.Pod` structure
   5. The `api.Pod` is converted to a `v5.Pod` structure
   6. The `v5.Pod` is marshalled into JSON and sent to the user

The implication of this process is that API changes must be done carefully and
backward-compatibly.

## On compatibility

Before talking about how to make API changes, it is worthwhile to clarify what
we mean by API compatibility.  An API change is considered backward-compatible
if it:
   * adds new functionality that is not required for correct behavior (e.g.,
     does not add a new required field)
   * does not change existing semantics, including:
     * default values and behavior
     * interpretation of existing API types, fields, and values
     * which fields are required and which are not

Put another way:

1. Any API call (e.g. a structure POSTed to a REST endpoint) that worked before
   your change must work the same after your change.
2. Any API call that uses your change must not cause problems (e.g. crash or
   degrade behavior) when issued against servers that do not include your change.
3. It must be possible to round-trip your change (convert to different API
   versions and back) with no loss of information.
4. Existing clients need not be aware of your change in order for them to continue
   to function as they did previously, even when your change is utilized

If your change does not meet these criteria, it is not considered strictly
compatible.

Let's consider some examples.  In a hypothetical API (assume we're at version
v6), the `Frobber` struct looks something like this:

```go
// API v6.
type Frobber struct {
	Height int    `json:"height"`
	Param  string `json:"param"`
}
```

You want to add a new `Width` field.  It is generally safe to add new fields
without changing the API version, so you can simply change it to:

```go
// Still API v6.
type Frobber struct {
	Height int    `json:"height"`
	Width  int    `json:"width"`
	Param  string `json:"param"`
}
```

The onus is on you to define a sane default value for `Width` such that rule #1
above is true - API calls and stored objects that used to work must continue to
work.

For your next change you want to allow multiple `Param` values.  You can not
simply change `Param string` to `Params []string` (without creating a whole new
API version) - that fails rules #1 and #2.  You can instead do something like:

```go
// Still API v6, but kind of clumsy.
type Frobber struct {
	Height int           `json:"height"`
	Width  int           `json:"width"`
	Param  string        `json:"param"`  // the first param
	ExtraParams []string `json:"params"` // additional params
}
```

Now you can satisfy the rules: API calls that provide the old style `Param`
will still work, while servers that don't understand `ExtraParams` can ignore
it.  This is somewhat unsatisfying as an API, but it is strictly compatible.

Part of the reason for versioning APIs and for using internal structs that are
distinct from any one version is to handle growth like this.  The internal
representation can be implemented as:

```go
// Internal, soon to be v7beta1.
type Frobber struct {
	Height int
	Width  int
	Params []string
}
```

The code that converts to/from versioned APIs can decode this into the somewhat
uglier (but compatible!) structures.  Eventually, a new API version, let's call
it v7beta1, will be forked and it can use the clean internal structure.

We've seen how to satisfy rules #1 and #2.  Rule #3 means that you can not
extend one versioned API without also extending the others.  For example, an
API call might POST an object in API v7beta1 format, which uses the cleaner
`Params` field, but the API server might store that object in trusty old v6
form (since v7beta1 is "beta").  When the user reads the object back in the
v7beta1 API it would be unacceptable to have lost all but `Params[0]`.  This
means that, even though it is ugly, a compatible change must be made to the v6
API.

However, this is very challenging to do correctly. It often requires
multiple representations of the same information in the same API resource, which
need to be kept in sync in the event that either is changed. For example,
let's say you decide to rename a field within the same API version. In this case,
you add units to `height` and `width`. You implement this by adding duplicate
fields:

```go
type Frobber struct {
	Height         *int          `json:"height"`
	Width          *int          `json:"width"`
	HeightInInches *int          `json:"heightInInches"`
	WidthInInches  *int          `json:"widthInInches"`
}
```

You convert all of the fields to pointers in order to distinguish between unset and
set to 0, and then set each corresponding field from the other in the defaulting
pass (e.g., `heightInInches` from `height`, and vice versa), which runs just prior
to conversion. That works fine when the user creates a resource from a hand-written
configuration -- clients can write either field and read either field, but what about
creation or update from the output of GET, or update via PATCH (see
[In-place updates](../user-guide/managing-deployments.md#in-place-updates-of-resources))?
In this case, the two fields will conflict, because only one field would be updated
in the case of an old client that was only aware of the old field (e.g., `height`).

Say the client creates:

```json
{
  "height": 10,
  "width": 5
}
```

and GETs:

```json
{
  "height": 10,
  "heightInInches": 10,
  "width": 5,
  "widthInInches": 5
}
```

then PUTs back:

```json
{
  "height": 13,
  "heightInInches": 10,
  "width": 5,
  "widthInInches": 5
}
```

The update should not fail, because it would have worked before `heightInInches` was added.

Therefore, when there are duplicate fields, the old field MUST take precedence
over the new, and the new field should be set to match by the server upon write.
A new client would be aware of the old field as well as the new, and so can ensure
that the old field is either unset or is set consistently with the new field. However,
older clients would be unaware of the new field. Please avoid introducing duplicate
fields due to the complexity they incur in the API.

A new representation, even in a new API version, that is more expressive than an old one
breaks backward compatibility, since clients that only understood the old representation
would not be aware of the new representation nor its semantics. Examples of
proposals that have run into this challenge include [generalized label
selectors](http://issues.k8s.io/341) and [pod-level security
context](http://prs.k8s.io/12823).

As another interesting example, enumerated values cause similar challenges.
Adding a new value to an enumerated set is *not* a compatible change.  Clients
which assume they know how to handle all possible values of a given field will
not be able to handle the new values.  However, removing value from an
enumerated set *can* be a compatible change, if handled properly (treat the
removed value as deprecated but allowed). This is actually a special case of
a new representation, discussed above.

## Incompatible API changes

There are times when this might be OK, but mostly we want changes that
meet this definition.  If you think you need to break compatibility,
you should talk to the Kubernetes team first.

Breaking compatibility of a beta or stable API version, such as v1, is unacceptable.
Compatibility for experimental or alpha APIs is not strictly required, but
breaking compatibility should not be done lightly, as it disrupts all users of the
feature. Experimental APIs may be removed. Alpha and beta API versions may be deprecated
and eventually removed wholesale, as described in the [versioning document](../design/versioning.md).
Document incompatible changes across API versions under the [conversion tips](../api.md).

If your change is going to be backward incompatible or might be a breaking change for API
consumers, please send an announcement to `kubernetes-dev@googlegroups.com` before
the change gets in. If you are unsure, ask. Also make sure that the change gets documented in
the release notes for the next release by labeling the PR with the "release-note" github label.

If you found that your change accidentally broke clients, it should be reverted.

In short, the expected API evolution is as follows:
* `experimental/v1alpha1` ->
* `newapigroup/v1alpha1` -> ... -> `newapigroup/v1alphaN` ->
* `newapigroup/v1beta1` -> ... -> `newapigroup/v1betaN` ->
* `newapigroup/v1` ->
* `newapigroup/v2alpha1` -> ...

While in experimental we have no obligation to move forward with the API at all and may delete or break it at any time.

While in alpha we expect to move forward with it, but may break it.

Once in beta we will preserve forward compatibility, but may introduce new versions and delete old ones.

v1 must be backward-compatible for an extended length of time.

## Changing versioned APIs

For most changes, you will probably find it easiest to change the versioned
APIs first.  This forces you to think about how to make your change in a
compatible way.  Rather than doing each step in every version, it's usually
easier to do each versioned API one at a time, or to do all of one version
before starting "all the rest".

### Edit types.go

The struct definitions for each API are in `pkg/api/<version>/types.go`.  Edit
those files to reflect the change you want to make.  Note that all types and non-inline
fields in versioned APIs must be preceded by descriptive comments - these are used to generate
documentation.

Optional fields should have the `,omitempty` json tag; fields are interpreted as being
required otherwise.

### Edit defaults.go

If your change includes new fields for which you will need default values, you
need to add cases to `pkg/api/<version>/defaults.go`.  Of course, since you
have added code, you have to add a test: `pkg/api/<version>/defaults_test.go`.

Do use pointers to scalars when you need to distinguish between an unset value
and an automatic zero value.  For example,
`PodSpec.TerminationGracePeriodSeconds` is defined as `*int64` the go type
definition.  A zero value means 0 seconds, and a nil value asks the system to
pick a default.

Don't forget to run the tests!

### Edit conversion.go

Given that you have not yet changed the internal structs, this might feel
premature, and that's because it is.  You don't yet have anything to convert to
or from.  We will revisit this in the "internal" section.  If you're doing this
all in a different order (i.e. you started with the internal structs), then you
should jump to that topic below.  In the very rare case that you are making an
incompatible change you might or might not want to do this now, but you will
have to do more later.  The files you want are
`pkg/api/<version>/conversion.go` and `pkg/api/<version>/conversion_test.go`.

Note that the conversion machinery doesn't generically handle conversion of values,
such as various kinds of field references and API constants. [The client
library](../../pkg/client/unversioned/request.go) has custom conversion code for
field references. You also need to add a call to api.Scheme.AddFieldLabelConversionFunc
with a mapping function that understands supported translations.

## Changing the internal structures

Now it is time to change the internal structs so your versioned changes can be
used.

### Edit types.go

Similar to the versioned APIs, the definitions for the internal structs are in
`pkg/api/types.go`.  Edit those files to reflect the change you want to make.
Keep in mind that the internal structs must be able to express *all* of the
versioned APIs.

## Edit validation.go

Most changes made to the internal structs need some form of input validation.
Validation is currently done on internal objects in
`pkg/api/validation/validation.go`.  This validation is the one of the first
opportunities we have to make a great user experience - good error messages and
thorough validation help ensure that users are giving you what you expect and,
when they don't, that they know why and how to fix it.  Think hard about the
contents of `string` fields, the bounds of `int` fields and the
requiredness/optionalness of fields.

Of course, code needs tests - `pkg/api/validation/validation_test.go`.

## Edit version conversions

At this point you have both the versioned API changes and the internal
structure changes done.  If there are any notable differences - field names,
types, structural change in particular - you must add some logic to convert
versioned APIs to and from the internal representation.  If you see errors from
the `serialization_test`, it may indicate the need for explicit conversions.

Performance of conversions very heavily influence performance of apiserver.
Thus, we are auto-generating conversion functions that are much more efficient
than the generic ones (which are based on reflections and thus are highly
inefficient).

The conversion code resides with each versioned API. There are two files:
   - `pkg/api/<version>/conversion.go` containing manually written conversion
     functions
   - `pkg/api/<version>/conversion_generated.go` containing auto-generated
     conversion functions
   - `pkg/apis/experimental/<version>/conversion.go` containing manually written
     conversion functions
   - `pkg/apis/experimental/<version>/conversion_generated.go` containing
     auto-generated conversion functions

Since auto-generated conversion functions are using manually written ones,
those manually written should be named with a defined convention, i.e. a function
converting type X in pkg a to type Y in pkg b, should be named:
`convert_a_X_To_b_Y`.

Also note that you can (and for efficiency reasons should) use auto-generated
conversion functions when writing your conversion functions.

Once all the necessary manually written conversions are added, you need to
regenerate auto-generated ones. To regenerate them:
   - run

```sh
hack/update-generated-conversions.sh
```

If running the above script is impossible due to compile errors, the easiest
workaround is to comment out the code causing errors and let the script to
regenerate it. If the auto-generated conversion methods are not used by the
manually-written ones, it's fine to just remove the whole file and let the
generator to create it from scratch.

Unsurprisingly, adding manually written conversion also requires you to add tests to
`pkg/api/<version>/conversion_test.go`.

## Edit deep copy files

At this point you have both the versioned API changes and the internal
structure changes done.  You now need to generate code to handle deep copy
of your versioned api objects.

The deep copy code resides with each versioned API:
   - `pkg/api/<version>/deep_copy_generated.go` containing auto-generated copy functions
   - `pkg/apis/experimental/<version>/deep_copy_generated.go` containing auto-generated copy functions

To regenerate them:
   - run

```sh
hack/update-generated-deep-copies.sh
```

## Making a new API Group

This section is under construction, as we make the tooling completely generic.

At the moment, you'll have to make a new directory under pkg/apis/; copy the
directory structure from pkg/apis/experimental. Add the new group/version to all
of the hack/{verify,update}-generated-{deep-copy,conversions,swagger}.sh files
in the appropriate places--it should just require adding your new group/version
to a bash array. You will also need to make sure your new types are imported by
the generation commands (cmd/gendeepcopy/ & cmd/genconversion). These
instructions may not be complete and will be updated as we gain experience.

Adding API groups outside of the pkg/apis/ directory is not currently supported,
but is clearly desirable. The deep copy & conversion generators need to work by
parsing go files instead of by reflection; then they will be easy to point at
arbitrary directories: see issue [#13775](http://issue.k8s.io/13775).

## Update the fuzzer

Part of our testing regimen for APIs is to "fuzz" (fill with random values) API
objects and then convert them to and from the different API versions.  This is
a great way of exposing places where you lost information or made bad
assumptions.  If you have added any fields which need very careful formatting
(the test does not run validation) or if you have made assumptions such as
"this slice will always have at least 1 element", you may get an error or even
a panic from the `serialization_test`.  If so, look at the diff it produces (or
the backtrace in case of a panic) and figure out what you forgot.  Encode that
into the fuzzer's custom fuzz functions.  Hint: if you added defaults for a field,
that field will need to have a custom fuzz function that ensures that the field is
fuzzed to a non-empty value.

The fuzzer can be found in `pkg/api/testing/fuzzer.go`.

## Update the semantic comparisons

VERY VERY rarely is this needed, but when it hits, it hurts.  In some rare
cases we end up with objects (e.g. resource quantities) that have morally
equivalent values with different bitwise representations (e.g. value 10 with a
base-2 formatter is the same as value 0 with a base-10 formatter).  The only way
Go knows how to do deep-equality is through field-by-field bitwise comparisons.
This is a problem for us.

The first thing you should do is try not to do that.  If you really can't avoid
this, I'd like to introduce you to our semantic DeepEqual routine.  It supports
custom overrides for specific types - you can find that in `pkg/api/helpers.go`.

There's one other time when you might have to touch this: unexported fields.
You see, while Go's `reflect` package is allowed to touch unexported fields, us
mere mortals are not - this includes semantic DeepEqual.  Fortunately, most of
our API objects are "dumb structs" all the way down - all fields are exported
(start with a capital letter) and there are no unexported fields.  But sometimes
you want to include an object in our API that does have unexported fields
somewhere in it (for example, `time.Time` has unexported fields).  If this hits
you, you may have to touch the semantic DeepEqual customization functions.

## Implement your change

Now you have the API all changed - go implement whatever it is that you're
doing!

## Write end-to-end tests

Check out the [E2E docs](e2e-tests.md) for detailed information about how to write end-to-end
tests for your feature.

## Examples and docs

At last, your change is done, all unit tests pass, e2e passes, you're done,
right?  Actually, no.  You just changed the API.  If you are touching an
existing facet of the API, you have to try *really* hard to make sure that
*all* the examples and docs are updated.  There's no easy way to do this, due
in part to JSON and YAML silently dropping unknown fields.  You're clever -
you'll figure it out.  Put `grep` or `ack` to good use.

If you added functionality, you should consider documenting it and/or writing
an example to illustrate your change.

Make sure you update the swagger API spec by running:

```sh
hack/update-swagger-spec.sh
```

The API spec changes should be in a commit separate from your other changes.

## Adding new REST objects

TODO(smarterclayton): write this.


<!-- BEGIN MUNGE: GENERATED_ANALYTICS -->
[![Analytics](https://kubernetes-site.appspot.com/UA-36037335-10/GitHub/docs/devel/api_changes.md?pixel)]()
<!-- END MUNGE: GENERATED_ANALYTICS -->
