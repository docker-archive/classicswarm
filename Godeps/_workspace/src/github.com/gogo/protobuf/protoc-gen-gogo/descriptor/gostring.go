package google_protobuf

import fmt "fmt"
import strings "strings"
import github_com_gogo_protobuf_proto "github.com/gogo/protobuf/proto"
import sort "sort"
import strconv "strconv"
import reflect "reflect"

func (this *FileDescriptorSet) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.FileDescriptorSet{` +
		`File:` + fmt.Sprintf("%#v", this.File),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *FileDescriptorProto) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.FileDescriptorProto{` +
		`Name:` + valueToGoStringDescriptor(this.Name, "string"),
		`Package:` + valueToGoStringDescriptor(this.Package, "string"),
		`Dependency:` + fmt.Sprintf("%#v", this.Dependency),
		`PublicDependency:` + fmt.Sprintf("%#v", this.PublicDependency),
		`WeakDependency:` + fmt.Sprintf("%#v", this.WeakDependency),
		`MessageType:` + fmt.Sprintf("%#v", this.MessageType),
		`EnumType:` + fmt.Sprintf("%#v", this.EnumType),
		`Service:` + fmt.Sprintf("%#v", this.Service),
		`Extension:` + fmt.Sprintf("%#v", this.Extension),
		`Options:` + fmt.Sprintf("%#v", this.Options),
		`SourceCodeInfo:` + fmt.Sprintf("%#v", this.SourceCodeInfo),
		`Syntax:` + valueToGoStringDescriptor(this.Syntax, "string"),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *DescriptorProto) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.DescriptorProto{` +
		`Name:` + valueToGoStringDescriptor(this.Name, "string"),
		`Field:` + fmt.Sprintf("%#v", this.Field),
		`Extension:` + fmt.Sprintf("%#v", this.Extension),
		`NestedType:` + fmt.Sprintf("%#v", this.NestedType),
		`EnumType:` + fmt.Sprintf("%#v", this.EnumType),
		`ExtensionRange:` + fmt.Sprintf("%#v", this.ExtensionRange),
		`OneofDecl:` + fmt.Sprintf("%#v", this.OneofDecl),
		`Options:` + fmt.Sprintf("%#v", this.Options),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *DescriptorProto_ExtensionRange) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.DescriptorProto_ExtensionRange{` +
		`Start:` + valueToGoStringDescriptor(this.Start, "int32"),
		`End:` + valueToGoStringDescriptor(this.End, "int32"),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *FieldDescriptorProto) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.FieldDescriptorProto{` +
		`Name:` + valueToGoStringDescriptor(this.Name, "string"),
		`Number:` + valueToGoStringDescriptor(this.Number, "int32"),
		`Label:` + valueToGoStringDescriptor(this.Label, "google_protobuf.FieldDescriptorProto_Label"),
		`Type:` + valueToGoStringDescriptor(this.Type, "google_protobuf.FieldDescriptorProto_Type"),
		`TypeName:` + valueToGoStringDescriptor(this.TypeName, "string"),
		`Extendee:` + valueToGoStringDescriptor(this.Extendee, "string"),
		`DefaultValue:` + valueToGoStringDescriptor(this.DefaultValue, "string"),
		`OneofIndex:` + valueToGoStringDescriptor(this.OneofIndex, "int32"),
		`Options:` + fmt.Sprintf("%#v", this.Options),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *OneofDescriptorProto) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.OneofDescriptorProto{` +
		`Name:` + valueToGoStringDescriptor(this.Name, "string"),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *EnumDescriptorProto) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.EnumDescriptorProto{` +
		`Name:` + valueToGoStringDescriptor(this.Name, "string"),
		`Value:` + fmt.Sprintf("%#v", this.Value),
		`Options:` + fmt.Sprintf("%#v", this.Options),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *EnumValueDescriptorProto) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.EnumValueDescriptorProto{` +
		`Name:` + valueToGoStringDescriptor(this.Name, "string"),
		`Number:` + valueToGoStringDescriptor(this.Number, "int32"),
		`Options:` + fmt.Sprintf("%#v", this.Options),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *ServiceDescriptorProto) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.ServiceDescriptorProto{` +
		`Name:` + valueToGoStringDescriptor(this.Name, "string"),
		`Method:` + fmt.Sprintf("%#v", this.Method),
		`Options:` + fmt.Sprintf("%#v", this.Options),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *MethodDescriptorProto) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.MethodDescriptorProto{` +
		`Name:` + valueToGoStringDescriptor(this.Name, "string"),
		`InputType:` + valueToGoStringDescriptor(this.InputType, "string"),
		`OutputType:` + valueToGoStringDescriptor(this.OutputType, "string"),
		`Options:` + fmt.Sprintf("%#v", this.Options),
		`ClientStreaming:` + valueToGoStringDescriptor(this.ClientStreaming, "bool"),
		`ServerStreaming:` + valueToGoStringDescriptor(this.ServerStreaming, "bool"),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *FileOptions) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.FileOptions{` +
		`JavaPackage:` + valueToGoStringDescriptor(this.JavaPackage, "string"),
		`JavaOuterClassname:` + valueToGoStringDescriptor(this.JavaOuterClassname, "string"),
		`JavaMultipleFiles:` + valueToGoStringDescriptor(this.JavaMultipleFiles, "bool"),
		`JavaGenerateEqualsAndHash:` + valueToGoStringDescriptor(this.JavaGenerateEqualsAndHash, "bool"),
		`JavaStringCheckUtf8:` + valueToGoStringDescriptor(this.JavaStringCheckUtf8, "bool"),
		`OptimizeFor:` + valueToGoStringDescriptor(this.OptimizeFor, "google_protobuf.FileOptions_OptimizeMode"),
		`GoPackage:` + valueToGoStringDescriptor(this.GoPackage, "string"),
		`CcGenericServices:` + valueToGoStringDescriptor(this.CcGenericServices, "bool"),
		`JavaGenericServices:` + valueToGoStringDescriptor(this.JavaGenericServices, "bool"),
		`PyGenericServices:` + valueToGoStringDescriptor(this.PyGenericServices, "bool"),
		`Deprecated:` + valueToGoStringDescriptor(this.Deprecated, "bool"),
		`CcEnableArenas:` + valueToGoStringDescriptor(this.CcEnableArenas, "bool"),
		`UninterpretedOption:` + fmt.Sprintf("%#v", this.UninterpretedOption),
		`XXX_extensions: ` + extensionToGoStringDescriptor(this.XXX_extensions),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *MessageOptions) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.MessageOptions{` +
		`MessageSetWireFormat:` + valueToGoStringDescriptor(this.MessageSetWireFormat, "bool"),
		`NoStandardDescriptorAccessor:` + valueToGoStringDescriptor(this.NoStandardDescriptorAccessor, "bool"),
		`Deprecated:` + valueToGoStringDescriptor(this.Deprecated, "bool"),
		`MapEntry:` + valueToGoStringDescriptor(this.MapEntry, "bool"),
		`UninterpretedOption:` + fmt.Sprintf("%#v", this.UninterpretedOption),
		`XXX_extensions: ` + extensionToGoStringDescriptor(this.XXX_extensions),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *FieldOptions) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.FieldOptions{` +
		`Ctype:` + valueToGoStringDescriptor(this.Ctype, "google_protobuf.FieldOptions_CType"),
		`Packed:` + valueToGoStringDescriptor(this.Packed, "bool"),
		`Lazy:` + valueToGoStringDescriptor(this.Lazy, "bool"),
		`Deprecated:` + valueToGoStringDescriptor(this.Deprecated, "bool"),
		`Weak:` + valueToGoStringDescriptor(this.Weak, "bool"),
		`UninterpretedOption:` + fmt.Sprintf("%#v", this.UninterpretedOption),
		`XXX_extensions: ` + extensionToGoStringDescriptor(this.XXX_extensions),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *EnumOptions) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.EnumOptions{` +
		`AllowAlias:` + valueToGoStringDescriptor(this.AllowAlias, "bool"),
		`Deprecated:` + valueToGoStringDescriptor(this.Deprecated, "bool"),
		`UninterpretedOption:` + fmt.Sprintf("%#v", this.UninterpretedOption),
		`XXX_extensions: ` + extensionToGoStringDescriptor(this.XXX_extensions),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *EnumValueOptions) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.EnumValueOptions{` +
		`Deprecated:` + valueToGoStringDescriptor(this.Deprecated, "bool"),
		`UninterpretedOption:` + fmt.Sprintf("%#v", this.UninterpretedOption),
		`XXX_extensions: ` + extensionToGoStringDescriptor(this.XXX_extensions),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *ServiceOptions) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.ServiceOptions{` +
		`Deprecated:` + valueToGoStringDescriptor(this.Deprecated, "bool"),
		`UninterpretedOption:` + fmt.Sprintf("%#v", this.UninterpretedOption),
		`XXX_extensions: ` + extensionToGoStringDescriptor(this.XXX_extensions),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *MethodOptions) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.MethodOptions{` +
		`Deprecated:` + valueToGoStringDescriptor(this.Deprecated, "bool"),
		`UninterpretedOption:` + fmt.Sprintf("%#v", this.UninterpretedOption),
		`XXX_extensions: ` + extensionToGoStringDescriptor(this.XXX_extensions),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *UninterpretedOption) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.UninterpretedOption{` +
		`Name:` + fmt.Sprintf("%#v", this.Name),
		`IdentifierValue:` + valueToGoStringDescriptor(this.IdentifierValue, "string"),
		`PositiveIntValue:` + valueToGoStringDescriptor(this.PositiveIntValue, "uint64"),
		`NegativeIntValue:` + valueToGoStringDescriptor(this.NegativeIntValue, "int64"),
		`DoubleValue:` + valueToGoStringDescriptor(this.DoubleValue, "float64"),
		`StringValue:` + valueToGoStringDescriptor(this.StringValue, "byte"),
		`AggregateValue:` + valueToGoStringDescriptor(this.AggregateValue, "string"),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *UninterpretedOption_NamePart) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.UninterpretedOption_NamePart{` +
		`NamePart:` + valueToGoStringDescriptor(this.NamePart, "string"),
		`IsExtension:` + valueToGoStringDescriptor(this.IsExtension, "bool"),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *SourceCodeInfo) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.SourceCodeInfo{` +
		`Location:` + fmt.Sprintf("%#v", this.Location),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func (this *SourceCodeInfo_Location) GoString() string {
	if this == nil {
		return "nil"
	}
	s := strings.Join([]string{`&google_protobuf.SourceCodeInfo_Location{` +
		`Path:` + fmt.Sprintf("%#v", this.Path),
		`Span:` + fmt.Sprintf("%#v", this.Span),
		`LeadingComments:` + valueToGoStringDescriptor(this.LeadingComments, "string"),
		`TrailingComments:` + valueToGoStringDescriptor(this.TrailingComments, "string"),
		`XXX_unrecognized:` + fmt.Sprintf("%#v", this.XXX_unrecognized) + `}`}, ", ")
	return s
}
func valueToGoStringDescriptor(v interface{}, typ string) string {
	rv := reflect.ValueOf(v)
	if rv.IsNil() {
		return "nil"
	}
	pv := reflect.Indirect(rv).Interface()
	return fmt.Sprintf("func(v %v) *%v { return &v } ( %#v )", typ, typ, pv)
}
func extensionToGoStringDescriptor(e map[int32]github_com_gogo_protobuf_proto.Extension) string {
	if e == nil {
		return "nil"
	}
	s := "map[int32]proto.Extension{"
	keys := make([]int, 0, len(e))
	for k := range e {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	ss := []string{}
	for _, k := range keys {
		ss = append(ss, strconv.Itoa(k)+": "+e[int32(k)].GoString())
	}
	s += strings.Join(ss, ",") + "}"
	return s
}
