package menmos

import (
	"fmt"
	"io"
)

type rangeReader struct {
	BlobID string
	Client *Client

	RangeStart int64
	RangeEnd   int64
}

func (r *rangeReader) Read(buf []byte) (int, error) {
	if r.RangeStart > r.RangeEnd {
		return 0, io.EOF
	}

	requestedDataLength := int64(len(buf))
	remainingRangeLength := (r.RangeEnd - r.RangeStart) + 1

	var lengthToRead int64
	if requestedDataLength <= remainingRangeLength {
		lengthToRead = requestedDataLength
	} else {
		lengthToRead = remainingRangeLength
	}

	rangeEnd := (r.RangeStart + lengthToRead) - 1

	responseReader, err := r.Client.readRange(r.BlobID, r.RangeStart, rangeEnd)
	if err != nil {
		return 0, err
	}
	defer responseReader.Close()

	readCount, err := io.ReadFull(responseReader, buf[:lengthToRead])
	if err != nil {
		return 0, err
	}

	if readCount != int(lengthToRead) {
		return 0, fmt.Errorf("range read returned incorrect amount of bytes: expected %d, got %d", lengthToRead, readCount)
	}

	r.RangeStart += int64(readCount)
	return readCount, nil
}

func (r *rangeReader) Close() error {
	return nil
}
