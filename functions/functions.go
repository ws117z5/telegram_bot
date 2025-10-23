package functions

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
)

func AssertEqual[T comparable](t *testing.T, expected T, actual T) {
	t.Helper()
	if expected == actual {
		return
	}
	t.Errorf("expected (%+v) is not equal to actual (%+v)", expected, actual)
}

func AssertSliceEqual[T comparable](t *testing.T, expected []T, actual []T) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Errorf("expected (%+v) is not equal to actual (%+v): len(expected)=%d len(actual)=%d",
			expected, actual, len(expected), len(actual))
	}
	for i := range expected {
		if expected[i] != actual[i] {
			t.Errorf("expected[%d] (%+v) is not equal to actual[%d] (%+v)", i, expected[i], i, actual[i])
		}
	}
}

func Dump(class interface{}) {
	JSON, _ := json.Marshal(class)
	Description, _ := json.MarshalIndent(JSON, "", "\t")
	fmt.Println(string(Description))
}

// readLines reads a whole file into memory
// and returns a slice of its lines.
func ReadLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// writeLines writes the lines to the given file.
func WriteLines(lines []string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}

func Map[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}

// Values returns the values of the map m.
// The values will be in an indeterminate order.
func Values[M ~map[K]V, K comparable, V any](m M) []V {
	r := make([]V, 0, len(m))
	for _, v := range m {
		r = append(r, v)
	}
	return r
}

func If[T any](cond bool, vtrue, vfalse T) T {
	if cond {
		return vtrue
	}
	return vfalse
}

func MapUsernames(name string) string {
	return "@" + name
}

type QueueChan[T any] struct {
	data  chan T
	size  int
	cap   int
	mutex *sync.Mutex
}

func (q *QueueChan[T]) Put(val T) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.cap >= q.size {
		return errors.New("Overflow")
	}

	q.cap++
	q.data <- val

	return nil
}

func (q *QueueChan[T]) Pop() (T, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.cap == 0 {
		return *new(T), errors.New("Empty queue")
	} else {
		q.cap--
		return <-q.data, nil
	}
}

func (q QueueChan[T]) Empty() bool {
	return q.cap == 0
}

func MakeQueueChan[T any](size int) QueueChan[T] {
	return QueueChan[T]{make(chan T, size), size, 0, &sync.Mutex{}}
}

type PriorityQueueQ[T any] struct {
	queue      chan T
	size       int
	cap        int
	comparator func(T, T) bool
	mutex      *sync.Mutex
}

func (this *PriorityQueueQ[T]) Put(data T) error {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	temp := make(chan T, this.size)

	close(this.queue)

	inserted := false
	for v := range this.queue {
		if !inserted && this.comparator(data, v) {
			temp <- data
			inserted = true
		}

		temp <- v
	}

	if !inserted {
		temp <- data
	}

	this.cap++

	this.queue = temp
	return nil
}

func (this *PriorityQueueQ[T]) Pop() (T, error) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.cap == 0 {
		return *new(T), errors.New("Empty queue")
	} else {
		this.cap--
		return <-this.queue, nil
	}

}

func (this *PriorityQueueQ[T]) Empty() bool {
	return this.cap == 0
}

func MakePriorityQueue[T any](size int, comparator func(T, T) bool) PriorityQueueQ[T] {
	return PriorityQueueQ[T]{make(chan T, size), size, 0, comparator, &sync.Mutex{}}
}
