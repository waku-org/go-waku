package cliutils

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/exp/slices"
)

type BridgeTopic struct {
	FromTopic string
	ToTopic   string
}

func (p BridgeTopic) String() string {
	return fmt.Sprintf("%s:%s", p.FromTopic, p.ToTopic)
}

type BridgeTopicSlice struct {
	Values *[]BridgeTopic
}

func (k *BridgeTopicSlice) Set(value string) error {
	topicParts := strings.Split(value, ":")
	if len(topicParts) != 2 {
		return errors.New("expected from_topic:to_topic")
	}

	for i := range topicParts {
		topicParts[i] = strings.TrimSpace(topicParts[i])
	}

	if slices.Contains(topicParts, "") {
		return errors.New("topic can't be empty")
	}

	*k.Values = append(*k.Values, BridgeTopic{
		FromTopic: topicParts[0],
		ToTopic:   topicParts[1],
	})

	return nil
}

func (k *BridgeTopicSlice) String() string {
	if k.Values == nil {
		return ""
	}
	var output []string
	for _, v := range *k.Values {
		output = append(output, v.String())
	}

	return strings.Join(output, ", ")
}
