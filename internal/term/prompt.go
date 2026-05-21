package term

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

func Line(in io.Reader, errOut io.Writer, label, defaultValue string) (string, error) {
	if in == nil {
		in = os.Stdin
	}
	if defaultValue != "" {
		fmt.Fprintf(errOut, "%s [%s]: ", label, defaultValue)
	} else {
		fmt.Fprintf(errOut, "%s: ", label)
	}
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultValue, nil
	}
	return line, nil
}

func Password(errOut io.Writer, label string) (string, error) {
	fmt.Fprintf(errOut, "%s: ", label)
	bytes, err := term.ReadPassword(int(os.Stdin.Fd())) //nolint:gosec // stdin fd is small on supported platforms
	fmt.Fprintln(errOut)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytes)), nil
}
