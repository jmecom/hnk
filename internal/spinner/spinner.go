package spinner

import (
	"fmt"
	"io"
	"sync"
	"time"
)

type Spinner struct {
	out     io.Writer
	message string
	frames  []string
	stop    chan struct{}
	done    chan struct{}
	mu      sync.Mutex
}

func New(out io.Writer, message string) *Spinner {
	return &Spinner{
		out:     out,
		message: message,
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
}

func (s *Spinner) Start() {
	go func() {
		defer close(s.done)
		i := 0
		for {
			select {
			case <-s.stop:
				s.clear()
				return
			default:
				s.mu.Lock()
				fmt.Fprintf(s.out, "\r%s %s", s.frames[i%len(s.frames)], s.message)
				s.mu.Unlock()
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

func (s *Spinner) Stop() {
	close(s.stop)
	<-s.done
}

func (s *Spinner) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Fprintf(s.out, "\r%*s\r", len(s.message)+3, "")
}
