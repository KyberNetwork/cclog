package common

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
)

type ConnectRequest struct {
	Name string
}

type ConnectResponse struct {
	Success bool
	Status  string
}

func encodeMessage(data interface{}) ([]byte, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	dataLength := len(b)
	if dataLength > math.MaxUint16 {
		return nil, errors.New("data too long")
	}
	framedData := make([]byte, len(b)+2)
	binary.LittleEndian.PutUint16(framedData, uint16(dataLength))
	n := copy(framedData[2:], b)
	if n != dataLength {
		return nil, errors.New("copy failed")
	}
	return framedData, nil
}

func WriteConnectRequest(w io.Writer, request ConnectRequest) error {
	data, err := encodeMessage(request)
	if err != nil {
		return err
	}
	_, _ = w.Write([]byte{'C', 'L'})
	_, err = w.Write(data)
	return err
}
func WriteConnectResponse(w io.Writer, request ConnectResponse) error {
	data, err := encodeMessage(request)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func readMessage(in io.Reader, out interface{}) error {
	var lengthBuff = [2]byte{}
	_, err := io.ReadFull(in, lengthBuff[:])
	if err != nil {
		return err
	}
	length := binary.LittleEndian.Uint16(lengthBuff[:])
	message := make([]byte, length)
	_, err = io.ReadFull(in, message)
	if err != nil {
		return fmt.Errorf("read message with length %d - %w", length, err)
	}
	err = json.Unmarshal(message, out)
	if err != nil {
		return fmt.Errorf("parse message failed, %s - %w", string(message), err)
	}
	return nil
}

func ReadConnectRequest(in io.Reader) (ConnectRequest, error) {
	var out ConnectRequest
	marker := [2]byte{}
	_, err := io.ReadFull(in, marker[:])
	if err != nil {
		return ConnectRequest{}, err
	}
	if marker[0] != 'C' || marker[1] != 'L' {
		return ConnectRequest{}, fmt.Errorf("close due client sent invalid header")
	}
	err = readMessage(in, &out)
	return out, err
}

func ReadConnectResponse(in io.Reader) (ConnectResponse, error) {
	var out ConnectResponse
	err := readMessage(in, &out)
	return out, err
}
