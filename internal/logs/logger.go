package logs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Logger struct {
	target writer

	c chan string

	traceID atomic.Uint32
}

type writer interface {
	Writeln(line string) error
}

type nopWriter struct{}

func (nopWriter) Writeln(_ string) error {
	return nil
}

func New(target writer) *Logger {
	if target == nil {
		target = nopWriter{}
	}

	out := &Logger{
		target: target,
		c:      make(chan string),
	}

	go out.start()

	return out
}

func (l *Logger) start() {
	for {
		str, ok := <-l.c
		if !ok {
			return
		}

		fmt.Println(str)

		if err := l.target.Writeln(str); err != nil {
			fmt.Printf("Logger: %v\n", err)
		}
	}
}

type Trace struct {
	id    string
	c     chan string
	start time.Time
}

func (l *Logger) Trace(name string) *Trace {
	l.traceID.Add(1)

	return &Trace{
		id: fmt.Sprintf("%03d %s", l.traceID.Load(), name),
		c:  l.c,
	}
}

func Var(name string, value any) string {
	switch v := value.(type) {
	case string, fmt.Stringer:
		return fmt.Sprintf("%s=%q", name, v)
	default:
		return fmt.Sprintf("%s=%v", name, value)
	}
}

func (t *Trace) Begins(vars ...string) *Trace {
	t.start = time.Now()

	str := t.id + " started"
	defer func() {
		t.c <- str
	}()

	if len(vars) == 0 {
		return t
	}

	str = fmt.Sprintf("%s [%s]", str, strings.Join(vars, ", "))

	return t
}

func (t *Trace) Ends(err error, vars ...string) {
	str := t.id + " finished in " + time.Since(t.start).String()
	defer func() {
		t.c <- str
	}()

	var v string
	if len(vars) != 0 {
		v = v + strings.Join(vars, ", ") + ", "
	}

	var e string
	if err != nil {
		e = err.Error()
	}

	str = fmt.Sprintf("%s [%serr=%q]", str, v, e)
}

func (t *Trace) Dump(name string, bytes []byte) {
	t.c <- fmt.Sprintf("%s %s dump\n%s", t.id, name, bytes)
}

type File struct {
	f  *os.File
	mu sync.Mutex
}

func NewFile(name string) (*File, error) {
	f, err := os.OpenFile(filepath.Clean(name)+".log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}

	return &File{f: f}, nil
}

func (f *File) Writeln(line string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	ln := fmt.Sprintf("%s %s\n", time.Now().Format("15:04:05.000"), line)
	_, err := f.f.WriteString(ln)

	return err
}

func (f *File) Close() {
	if err := f.f.Close(); err != nil {
		fmt.Printf("Close: %v\n", err)
	}
}
