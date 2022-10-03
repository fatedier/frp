package port

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	NameDelimiter = "_"
)

type NameOption func(*nameBuilder) *nameBuilder

type nameBuilder struct {
	name          string
	rangePortFrom int
	rangePortTo   int
}

func unmarshalFromName(name string) (*nameBuilder, error) {
	var builder nameBuilder
	arrs := strings.Split(name, NameDelimiter)
	switch len(arrs) {
	case 2:
		builder.name = arrs[1]
	case 4:
		builder.name = arrs[1]
		fromPort, err := strconv.Atoi(arrs[2])
		if err != nil {
			return nil, fmt.Errorf("error range port from")
		}
		builder.rangePortFrom = fromPort

		toPort, err := strconv.Atoi(arrs[3])
		if err != nil {
			return nil, fmt.Errorf("error range port to")
		}
		builder.rangePortTo = toPort
	default:
		return nil, fmt.Errorf("error port name format")
	}
	return &builder, nil
}

func (builder *nameBuilder) String() string {
	name := fmt.Sprintf("Port%s%s", NameDelimiter, builder.name)
	if builder.rangePortFrom > 0 && builder.rangePortTo > 0 && builder.rangePortTo > builder.rangePortFrom {
		name += fmt.Sprintf("%s%d%s%d", NameDelimiter, builder.rangePortFrom, NameDelimiter, builder.rangePortTo)
	}
	return name
}

func WithRangePorts(from, to int) NameOption {
	return func(builder *nameBuilder) *nameBuilder {
		builder.rangePortFrom = from
		builder.rangePortTo = to
		return builder
	}
}

func GenName(name string, options ...NameOption) string {
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "_", "")
	builder := &nameBuilder{name: name}
	for _, option := range options {
		builder = option(builder)
	}
	return builder.String()
}
