package assert

// Copyright (c) 2017 Blake Williams <code@shabbyrobe.org>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func WrapTB(tb testing.TB) T { tb.Helper(); return T{TB: tb} }

// T wraps a testing.T or a testing.B with a simple set of custom assertions.
//
// Assertions prefixed with 'Must' will terminate the execution of the test case
// immediately.
//
// Assertions that are not prefixed with 'Must' will fail the test but allow
// the test to continue.
type T struct{ testing.TB }

// frameDepth is the number of frames to strip off the callstack when reporting the line
// where an error occurred.
const frameDepth = 2

func CompareMsg(exp, act interface{}) string {
	return fmt.Sprintf("\nexp: %+v\ngot: %+v", exp, act)
}

func CompareMsgf(exp, act interface{}, msg string, args ...interface{}) string {
	msg = fmt.Sprintf(msg, args...)
	return fmt.Sprintf("%v%v", msg, CompareMsg(exp, act))
}

// MustAssert immediately fails the test if the condition is false.
func (tb T) MustAssert(condition bool, v ...interface{}) {
	tb.Helper()
	_ = tb.assert(true, condition, v...)
}

// Assert fails the test if the condition is false.
func (tb T) Assert(condition bool, v ...interface{}) bool {
	tb.Helper()
	return tb.assert(false, condition, v...)
}

func (tb T) assert(fatal bool, condition bool, v ...interface{}) bool {
	tb.Helper()
	if !condition {
		_, file, line, _ := runtime.Caller(frameDepth)
		msg := ""
		if len(v) > 0 {
			msgx := v[0]
			v = v[1:]
			if msgx == nil {
				msg = "<nil>"
			} else if err, ok := msgx.(error); ok {
				msg = err.Error()
			} else {
				msg = msgx.(string)
			}
		}
		v = append([]interface{}{filepath.Base(file), line}, v...)
		tb.fail(fatal, fmt.Sprintf("\nassertion failed at %s:%d\n"+msg, v...))
	}
	return condition
}

// MustOKAll errors and terminates the test at the first error found in the arguments.
// It allows multiple return value functions to be passed in directly.
func (tb T) MustOKAll(errs ...interface{}) {
	tb.Helper()
	_ = tb.okAll(true, errs...)
}

// OKAll errors the test at the first error found in the arguments, but continues
// running the test. It allows multiple return value functions to be passed in
// directly.
func (tb T) OKAll(errs ...interface{}) bool {
	tb.Helper()
	return tb.okAll(false, errs...)
}

func (tb T) okAll(fatal bool, errs ...interface{}) bool {
	tb.Helper()
	for _, err := range errs {
		if _, ok := err.(*testing.T); ok {
			panic("unexpected testing.T in call to OK()")
		} else if _, ok := err.(T); ok {
			panic("unexpected testtools.T in call to OK()")
		}
		if err, ok := err.(error); ok && err != nil {
			if !tb.ok(fatal, err) {
				return false
			}
		}
	}
	return true
}

func (tb T) MustOK(err error) {
	tb.Helper()
	_ = tb.ok(true, err)
}

func (tb T) OK(err error) bool {
	tb.Helper()
	return tb.ok(true, err)
}

func (tb T) ok(fatal bool, err error) bool {
	tb.Helper()
	if err == nil {
		return true
	}
	_, file, line, _ := runtime.Caller(frameDepth)
	tb.fail(fatal, fmt.Sprintf("\nunexpected error at %s:%d\n%s",
		filepath.Base(file), line, err.Error()))
	return false
}

// MustExact immediately fails the test if the Go language equality rules for
// '==' do not apply to the arguments. This is distinct from MustEqual, which
// performs a reflect.DeepEqual().
//
func (tb T) MustExact(exp, act interface{}, v ...interface{}) {
	tb.Helper()
	_ = tb.exact(true, exp, act, v...)
}

// Exact fails the test but continues executing if the Go language equality
// rules for '==' do not apply to the arguments. This is distinct from
// MustEqual, which performs a reflect.DeepEqual().
//
func (tb T) Exact(exp, act interface{}, v ...interface{}) bool {
	tb.Helper()
	return tb.exact(false, exp, act, v...)
}

func (tb T) exact(fatal bool, exp, act interface{}, v ...interface{}) bool {
	tb.Helper()
	if exp != act {
		tb.failCompare("exact", exp, act, fatal, frameDepth+1, v...)
		return false
	}
	return true
}

// MustEqual immediately fails the test if exp is not equal to act based on
// reflect.DeepEqual(). See Exact for equality comparisons using '=='.
func (tb T) MustEqual(exp, act interface{}, v ...interface{}) {
	tb.Helper()
	_ = tb.equals(true, exp, act, v...)
}

// Equals fails the test but continues executing if exp is not equal to act
// using reflect.DeepEqual() and returns whether the assertion succeded. See
// Exact for equality comparisons using '=='.
func (tb T) Equals(exp, act interface{}, v ...interface{}) bool {
	tb.Helper()
	return tb.equals(false, exp, act, v...)
}

// Equal fails the test if exp is not equal to act.
func (tb T) equals(fatal bool, exp, act interface{}, v ...interface{}) bool {
	tb.Helper()
	if !reflect.DeepEqual(exp, act) {
		tb.failCompare("equal", exp, act, fatal, frameDepth+1, v...)
		return false
	}
	return true
}

func (tb T) MustFloatNear(epsilon float64, expected float64, actual float64, v ...interface{}) {
	tb.Helper()
	_ = tb.floatNear(true, epsilon, expected, actual, v...)
}

func (tb T) MustFloatsNear(epsilon float64, expected []float64, actual []float64, v ...interface{}) {
	tb.Helper()
	tb.MustEqual(len(expected), len(actual), "length mismatch")
	for i := range expected {
		_ = tb.floatNear(true, epsilon, expected[i], actual[i], v...)
	}
}

func (tb T) FloatNear(epsilon float64, expected float64, actual float64, v ...interface{}) bool {
	tb.Helper()
	return tb.floatNear(false, epsilon, expected, actual, v...)
}

func (tb T) floatNear(fatal bool, epsilon float64, expected float64, actual float64, v ...interface{}) bool {
	tb.Helper()
	near := IsFloatNear(epsilon, expected, actual)
	if !near {
		_, file, line, _ := runtime.Caller(frameDepth)
		msg := ""
		if len(v) > 0 {
			msg, v = v[0].(string), v[1:]
		}
		v = append([]interface{}{expected, actual, epsilon, filepath.Base(file), line}, v...)
		msg = fmt.Sprintf("\nfloat abs(%f - %f) > %f at %s:%d\n"+msg, v...)
		tb.fail(fatal, msg)
	}
	return near
}

func (tb T) failCompare(kind string, exp, act interface{}, fatal bool, frameOffset int, v ...interface{}) {
	tb.Helper()
	extra := ""
	if len(v) > 0 {
		extra = fmt.Sprintf(" - "+v[0].(string), v[1:]...)
	}

	_, file, line, _ := runtime.Caller(frameOffset)
	msg := CompareMsgf(exp, act, "\n%s failed at %s:%d%s", kind, filepath.Base(file), line, extra)
	tb.fail(fatal, msg)
}

func (tb T) fail(fatal bool, msg string) {
	tb.Helper()
	if fatal {
		tb.Fatal(msg)
	} else {
		tb.Error(msg)
	}
}

func IsFloatNear(epsilon, expected, actual float64) bool {
	diff := expected - actual
	return diff == 0 || (diff < 0 && diff > -epsilon) || (diff > 0 && diff < epsilon)
}
