package main

import (
	"encoding/json"
	"io"
	"os"
)

type record struct {
	Type string `json:"type"`
	Org  string `json:"org"`
	Repo string `json:"repo"`
	Id   string `json:"id"`
}

type serializable interface {
	Type() string
	Org() string
	Repo() string
	ID() string
}

func save(filename string, item serializable) error {
	dataFile, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer dataFile.Close()

	return nil
}

func remove(filename string, id string) error {
	return nil
}

func serialize(w io.Writer, items []serializable) error {
	var records []record

	for _, item := range items {
		records = append(records, record{
			Type: item.Type(),
			Org:  item.Org(),
			Repo: item.Repo(),
			Id:   item.ID(),
		})
	}

	encoder := json.NewEncoder(w)
	return encoder.Encode(records)
}

func deserializeRecords(r io.Reader) ([]record, error) {
	var records []record

	decoder := json.NewDecoder(r)
	err := decoder.Decode(&records)
	if err != nil {
		return nil, err
	}

	return records, nil
}
