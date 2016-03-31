package redis

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"unicode/utf8"
)

// connection ...
type connection struct {
	Con net.Conn
	Cmd bytes.Buffer
}

// Close close the connection to redis server
func (c *connection) Close() error {
	return c.Con.Close()
}

// Exec do a single command
func (c *connection) Exec(cmd string, args ...interface{}) Result {
	res := new(redisResult)

	err := c.writeCmd(cmd, args...)
	if err != nil {
		res.Res = err
		return res
	}
	err = c.flush()
	if err != nil {
		res.Res = err
		return res
	}
	return c.read()
}

//Pipline cache all the command
func (c *connection) Pipline(cmd string, args ...interface{}) error {
	return c.writeCmd(cmd, args...)
}

func (c *connection) Commit() Result {
	res := new(redisResult)

	err := c.flush()
	if err != nil {
		res.Res = err
		return res
	}
	return c.read()
}

func (c *connection) flush() error {
	defer c.clear()

	_, err := c.Con.Write(c.Cmd.Bytes())

	if err != nil {
		return err
	}
	return nil
}

func (c *connection) read() Result {
	res := &redisResult{}
	buf := make([]byte, 512)
	n, err := c.Con.Read(buf)
	if err != nil {
		res.Res = err
		return res
	}
	tmp := buf[:n]

	switch string(tmp[:1]) {
	case "+":
		str := parseResponse(string(tmp[1:]))
		res.Res = str

	case "-":
		err := parseError(string(tmp[1:]))
		res.Res = err

	case "$":
		str := parseSingleLineString(string(tmp[1:]))
		res.Res = str

	case "*":
		arr := parseArr(string(tmp[1:]))
		res.Res = arr

	case ":":
		num := parseInt(string(tmp[1:]))
		res.Res = num
	}
	return res
}

func parseResponse(s string) string {
	return strings.TrimRight(s, "\r\n")
}

func parseSingleLineString(s string) string {
	return strings.Split(s, "\r\n")[1]
}

func parseInt(s string) int {
	str := strings.TrimRight(s, "\r\n")
	i, _ := strconv.Atoi(str)
	return i
}

func parseArr(s string) []string {
	return strings.Split(s, "\r\n")[1:]
}

func parseError(s string) error {
	return errors.New(strings.TrimRight(s, "\r\n"))
}

func (c *connection) writeCmd(cmd string, args ...interface{}) (err error) {
	c.writeLength(len(args) + 1)
	c.writeString(cmd)
	for _, arg := range args {
		switch arg.(type) {
		case string:
			c.writeString(arg.(string))
		case int32:
			c.writeInt32(arg.(int32))
		case int64:
			c.writeInt64(arg.(int64))
		case []byte:
			c.writeBytes(arg.([]byte))
		case float32:
			c.writeFloat32(arg.(float32))
		case float64:
			c.writeFloat64(arg.(float64))
		default:
			panic(errors.New("unknow type"))
		}
	}
	return
}

func (c *connection) writeLength(length int) {
	str := fmt.Sprintf("*%d\r\n", length)
	_, err := c.Cmd.WriteString(str)
	if err != nil {
		panic(err)
	}
}

func (c *connection) writeString(s string) {
	str := fmt.Sprintf("$%d\r\n%s\r\n", utf8.RuneCountInString(s), s)

	_, err := c.Cmd.WriteString(str)
	if err != nil {
		panic(err)
	}
}

func (c *connection) writeBytes(bts []byte) {
	str := fmt.Sprintf("$%d\r\n%s\r\n", utf8.RuneCount(bts), bts)
	_, err := c.Cmd.WriteString(str)
	if err != nil {
		panic(err)
	}
}

func (c *connection) writeInt64(i int64) {
	str := fmt.Sprint(i)
	c.writeString(str)
}

func (c *connection) writeFloat64(f float64) {
	str := fmt.Sprint(f)
	c.writeString(str)
}

func (c *connection) writeInt32(i int32) {
	str := fmt.Sprint(i)
	c.writeString(str)
}

func (c *connection) writeFloat32(f float32) {
	str := fmt.Sprint(f)
	c.writeString(str)
}

func (c *connection) clear() {
	c.Cmd.Reset()
}