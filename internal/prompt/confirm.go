package prompt

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func ConfirmOverwrite(path string, in io.Reader, out io.Writer) (bool, error) {
	reader := bufio.NewReader(in)
	fmt.Fprintf(out, "File '%s' already exists. Overwrite? (y/N): ", path)
	reply, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return false, nil
	}
	return strings.EqualFold(reply, "y") || strings.EqualFold(reply, "yes"), nil
}
