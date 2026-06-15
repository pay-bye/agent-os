package processlog

import (
	"encoding/json"
	"io"
	"time"
)

type Clock interface {
	Now() time.Time
}

type Recorder interface {
	Record(Record)
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now().UTC()
}

type Sink struct {
	writer io.Writer
	clock  Clock
}

func NewSink(writer io.Writer, clock Clock) Sink {
	if writer == nil {
		writer = io.Discard
	}
	if clock == nil {
		clock = systemClock{}
	}
	return Sink{writer: writer, clock: clock}
}

func (s Sink) Record(record Record) {
	_ = s.Emit(record)
}

func (s Sink) Emit(record Record) error {
	if err := validate(record); err != nil {
		return err
	}
	record.Timestamp = s.clock.Now().UTC().Format(time.RFC3339)
	encoded, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if _, err := s.writer.Write(append(encoded, '\n')); err != nil {
		return err
	}
	return nil
}
