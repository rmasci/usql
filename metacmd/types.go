package metacmd

import (
	"context"
	"database/sql"
	"io"
	"os/user"
	"strings"
	"time"

	"github.com/xo/dburl"
	"github.com/rmasci/usql/drivers"
	"github.com/rmasci/usql/drivers/metadata"
	"github.com/rmasci/usql/env"
	"github.com/rmasci/usql/rline"
	"github.com/rmasci/usql/stmt"
	"github.com/rmasci/usql/text"
)

// Handler is the shared interface for a command handler.
type Handler interface {
	// IO handles the handler's IO.
	IO() rline.IO
	// User returns the current user.
	User() *user.User
	// URL returns the current database URL.
	URL() *dburl.URL
	// DB returns the current database connection.
	DB() drivers.DB
	// Last returns the last executed query.
	Last() string
	// LastRaw returns the last raw (non-interpolated) query.
	LastRaw() string
	// Buf returns the current query buffer.
	Buf() *stmt.Stmt
	// Reset resets the last and current query buffer.
	Reset([]rune)
	// Open opens a database connection.
	Open(context.Context, ...string) error
	// Close closes the current database connection.
	Close() error
	// ChangePassword changes the password for a user.
	ChangePassword(string) (string, error)
	// ReadVar reads a variable of a specified type.
	ReadVar(string, string) (string, error)
	// Include includes a file.
	Include(string, bool) error
	// Begin begins a transaction.
	Begin(*sql.TxOptions) error
	// Commit commits the current transaction.
	Commit() error
	// Rollback aborts the current transaction.
	Rollback() error
	// Highlight highlights the statement.
	Highlight(io.Writer, string) error
	// GetTiming mode.
	GetTiming() bool
	// SetTiming mode.
	SetTiming(bool)
	// GetOutput writer.
	GetOutput() io.Writer
	// SetOutput writer.
	SetOutput(io.WriteCloser)
	// MetadataWriter retrieves the metadata writer for the handler.
	MetadataWriter(context.Context) (metadata.Writer, error)
	// Print formats according to a format specifier and writes to handler's standard output.
	Print(string, ...interface{})
}

// Runner is a runner interface type.
type Runner interface {
	Run(Handler) (Option, error)
}

// RunnerFunc is a type wrapper for a single func satisfying Runner.Run.
type RunnerFunc func(Handler) (Option, error)

// Run satisfies the Runner interface.
func (f RunnerFunc) Run(h Handler) (Option, error) {
	return f(h)
}

// ExecType represents the type of execution requested.
type ExecType int

const (
	// ExecNone indicates no execution.
	ExecNone ExecType = iota
	// ExecOnly indicates plain execution only (\g).
	ExecOnly
	// ExecPipe indicates execution and piping results (\g |file)
	ExecPipe
	// ExecSet indicates execution and setting the resulting columns as
	// variables (\gset).
	ExecSet
	// ExecExec indicates execution and executing the resulting rows (\gexec).
	ExecExec
	// ExecCrosstab indicates execution using crosstabview (\crosstabview).
	ExecCrosstab
	// ExecWatch indicates repeated execution with a fixed time interval.
	ExecWatch
)

// Option contains parsed result options of a metacmd.
type Option struct {
	// Quit instructs the handling code to quit.
	Quit bool
	// Exec informs the handling code of the type of execution.
	Exec ExecType
	// Params are accompanying string parameters for execution.
	Params map[string]string
	// Crosstab are the crosstab column parameters.
	Crosstab []string
	// Watch is the watch duration interval.
	Watch time.Duration
}

func (opt *Option) ParseParams(params []string, defaultKey string) error {
	if opt.Params == nil {
		opt.Params = make(map[string]string, len(params))
	}
	formatOptions := false
	for i, param := range params {
		if len(param) == 0 {
			continue
		}
		if !formatOptions {
			if param[0] == '(' {
				formatOptions = true
			} else {
				opt.Params[defaultKey] = strings.Join(params[i:], " ")
				return nil
			}
		}
		parts := strings.SplitN(param, "=", 2)
		if len(parts) == 1 {
			return text.ErrInvalidFormatOption
		}
		opt.Params[strings.TrimLeft(parts[0], "(")] = strings.TrimRight(parts[1], ")")
		if formatOptions && param[len(param)-1] == ')' {
			formatOptions = false
		}
	}
	return nil
}

// Params wraps metacmd parameters.
type Params struct {
	// Handler is the process handler.
	Handler Handler
	// Name is the name of the metacmd.
	Name string
	// Params are the actual statement parameters.
	Params *stmt.Params
	// Option contains resulting command execution options.
	Option Option
}

// Get returns the next command parameter, using env.Unquote to decode quoted
// strings.
func (p *Params) Get(exec bool) (string, error) {
	_, v, err := p.Params.Get(env.Unquote(
		p.Handler.User(),
		exec,
		env.All(),
	))
	if err != nil {
		return "", err
	}
	return v, nil
}

// GetOK returns the next command parameter, using env.Unquote to decode quoted
// strings.
func (p *Params) GetOK(exec bool) (bool, string, error) {
	return p.Params.Get(env.Unquote(
		p.Handler.User(),
		exec,
		env.All(),
	))
}

// GetOptional returns the next command parameter, using env.Unquote to decode
// quoted strings, returns true when the value is prefixed with a "-", along
// with the value sans the "-" prefix. Otherwise returns false and the value.
func (p *Params) GetOptional(exec bool) (bool, string, error) {
	v, err := p.Get(exec)
	if err != nil {
		return false, "", err
	}
	if len(v) > 0 && v[0] == '-' {
		return true, v[1:], nil
	}
	return false, v, nil
}

// GetAll gets all remaining command parameters using env.Unquote to decode
// quoted strings.
func (p *Params) GetAll(exec bool) ([]string, error) {
	return p.Params.GetAll(env.Unquote(
		p.Handler.User(),
		exec,
		env.All(),
	))
}

// GetRaw gets the remaining command parameters as a raw string.
//
// Note: no other processing is done to interpolate variables or to decode
// string values.
func (p *Params) GetRaw() string {
	return p.Params.GetRaw()
}
