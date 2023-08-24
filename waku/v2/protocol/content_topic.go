package protocol

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const DefaultContentTopic = "/waku/2/default-content/proto"

var ErrInvalidFormat = errors.New("invalid format")
var ErrMissingGeneration = errors.New("missing part: generation")
var ErrInvalidGeneration = errors.New("generation should be a number")

type ContentTopic struct {
	ContentTopicParams
	ApplicationName    string
	ApplicationVersion uint32
	ContentTopicName   string
	Encoding           string
}

type ContentTopicParams struct {
	Generation int
}

func (ctp ContentTopicParams) Equal(ctp2 ContentTopicParams) bool {
	return ctp.Generation == ctp2.Generation
}

type ContentTopicOption func(*ContentTopicParams)

func (ct ContentTopic) String() string {
	return fmt.Sprintf("/%s/%d/%s/%s", ct.ApplicationName, ct.ApplicationVersion, ct.ContentTopicName, ct.Encoding)
}

func NewContentTopic(applicationName string, applicationVersion uint32,
	contentTopicName string, encoding string, opts ...ContentTopicOption) (ContentTopic, error) {

	params := new(ContentTopicParams)
	optList := DefaultOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		opt(params)
	}
	if params.Generation > 0 {
		return ContentTopic{}, ErrInvalidGeneration
	}
	return ContentTopic{
		ContentTopicParams: *params,
		ApplicationName:    applicationName,
		ApplicationVersion: applicationVersion,
		ContentTopicName:   contentTopicName,
		Encoding:           encoding,
	}, nil
}

func WithGeneration(generation int) ContentTopicOption {
	return func(params *ContentTopicParams) {
		params.Generation = generation
	}
}

func DefaultOptions() []ContentTopicOption {
	return []ContentTopicOption{
		WithGeneration(0),
	}
}

func (ct ContentTopic) Equal(ct2 ContentTopic) bool {
	return ct.ApplicationName == ct2.ApplicationName && ct.ApplicationVersion == ct2.ApplicationVersion &&
		ct.ContentTopicName == ct2.ContentTopicName && ct.Encoding == ct2.Encoding &&
		ct.ContentTopicParams.Equal(ct2.ContentTopicParams)
}

func StringToContentTopic(s string) (ContentTopic, error) {
	p := strings.Split(s, "/")
	switch len(p) {
	case 5:
		vNum, err := strconv.ParseUint(p[2], 10, 32)
		if err != nil {
			return ContentTopic{}, ErrInvalidFormat
		}

		return ContentTopic{
			ApplicationName:    p[1],
			ApplicationVersion: uint32(vNum),
			ContentTopicName:   p[3],
			Encoding:           p[4],
		}, nil
	case 6:
		if len(p[1]) == 0 {
			return ContentTopic{}, ErrMissingGeneration
		}
		generation, err := strconv.Atoi(p[1])
		if err != nil || generation > 0 {
			return ContentTopic{}, ErrInvalidGeneration
		}
		vNum, err := strconv.ParseUint(p[3], 10, 32)
		if err != nil {
			return ContentTopic{}, ErrInvalidFormat
		}

		return ContentTopic{
			ContentTopicParams: ContentTopicParams{Generation: generation},
			ApplicationName:    p[2],
			ApplicationVersion: uint32(vNum),
			ContentTopicName:   p[4],
			Encoding:           p[5],
		}, nil
	default:
		return ContentTopic{}, ErrInvalidFormat
	}
}
