package rax

import (
	"fmt"
	"io"
	"io/ioutil"
	"github.com/docker/libswarm/beam"
)

type rax_container struct {
	rax *raxcloud
	id  string
}

func (c *rax_container) attach(msg *beam.Message) error {
	var ctx *HostContext

	if _, err := msg.Ret.Send(&beam.Message{Verb: beam.Ack}); err != nil {
		return err
	} else if ctx, err = c.rax.getHostContext(); err == nil {
		return ctx.exec(msg, func(msg *beam.Message, client HttpClient) (err error) {
			path := fmt.Sprintf("/containers/%s/attach?stdout=1&stderr=1&stream=1", c.id)

			stdoutR, stdoutW := io.Pipe()
			stderrR, stderrW := io.Pipe()
			go copyOutput(msg.Ret, stdoutR, "stdout")
			go copyOutput(msg.Ret, stderrR, "stderr")

			client.Hijack("POST", path, nil, stdoutW, stderrW)

			return
		})
	} else {
		return fmt.Errorf("Failed to create host context: %v", err)
	}

	return nil
}

func (c *rax_container) start(msg *beam.Message) error {
	var ctx *HostContext

	if _, err := msg.Ret.Send(&beam.Message{Verb: beam.Ack}); err != nil {
		return err
	} else if ctx, err = c.rax.getHostContext(); err == nil {
		return ctx.exec(msg, func(msg *beam.Message, client HttpClient) (err error) {
			path := fmt.Sprintf("/containers/%s/start", c.id)
			resp, err := client.Post(path, "{}")
			if err != nil {
				return err
			}

			respBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if resp.StatusCode != 204 {
				return fmt.Errorf("expected status code 204, got %d:\n%s", resp.StatusCode, respBody)
			}

			if _, err := msg.Ret.Send(&beam.Message{Verb: beam.Ack}); err != nil {
				return err
			}

			return nil
		})
	} else {
		return fmt.Errorf("Failed to create host context: %v", err)
	}

	return nil
}

func (c *rax_container) stop(msg *beam.Message) error {
	var ctx *HostContext

	if _, err := msg.Ret.Send(&beam.Message{Verb: beam.Ack}); err != nil {
		return err
	} else if ctx, err = c.rax.getHostContext(); err == nil {
		return ctx.exec(msg, func(msg *beam.Message, client HttpClient) (err error) {
			path := fmt.Sprintf("/containers/%s/start", c.id)
			resp, err := client.Post(path, "{}")
			if err != nil {
				return err
			}

			respBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if resp.StatusCode != 204 {
				return fmt.Errorf("expected status code 204, got %d:\n%s", resp.StatusCode, respBody)
			}

			if _, err := msg.Ret.Send(&beam.Message{Verb: beam.Ack}); err != nil {
				return err
			}

			return nil
		})
	} else {
		return fmt.Errorf("Failed to create host context: %v", err)
	}

	return nil
}

func (c *rax_container) get(msg *beam.Message) error {
	var ctx *HostContext

	if _, err := msg.Ret.Send(&beam.Message{Verb: beam.Ack}); err != nil {
		return err
	} else if ctx, err = c.rax.getHostContext(); err == nil {
		return ctx.exec(msg, func(msg *beam.Message, client HttpClient) (err error) {
			path := fmt.Sprintf("/containers/%s/json", c.id)
			resp, err := client.Get(path, "")
			if err != nil {
				return err
			}

			respBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected status code 200, got %d:\n%s", resp.StatusCode, respBody)
			}

			if _, err := msg.Ret.Send(&beam.Message{Verb: beam.Set, Args: []string{string(respBody)}}); err != nil {
				return err
			}

			return nil
		})
	} else {
		return fmt.Errorf("Failed to create host context: %v", err)
	}

	return nil
}
