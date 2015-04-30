package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/pkg/term"
)

func readInput(in io.Reader, out io.Writer) string {
	reader := bufio.NewReader(in)
	line, _, err := reader.ReadLine()
	if err != nil {
		fmt.Fprintln(out, err.Error())
		os.Exit(1)
	}
	return string(line)
}

func getUsernameAndPassword(username, password string) (string, string, error) {
	if username == "" {
		fmt.Fprintf(os.Stdout, "Username: ")
		username = readInput(os.Stdin, os.Stdout)
		username = strings.Trim(username, " ")
	}

	if password == "" {
		oldState, err := term.SaveState(os.Stdin.Fd())
		if err != nil {
			return "", "", err
		}
		fmt.Fprintf(os.Stdout, "Password: ")
		term.DisableEcho(os.Stdin.Fd(), oldState)

		password = readInput(os.Stdin, os.Stdout)
		fmt.Fprint(os.Stdout, "\n")

		term.RestoreTerminal(os.Stdin.Fd(), oldState)
	}
	return username, password, nil
}
