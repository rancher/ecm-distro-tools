package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type record struct {
	Type   string `json:"type"`
	Org    string `json:"org"`
	Repo   string `json:"repo"`
	Id     string `json:"id"`
	Server string `json:"server"`
}

func serialize(filename string, items []listItem) error {
	records := make([]record, 0, len(items))
	for _, item := range items {
		records = append(records, item.Record())
	}

	jsonData, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		return err
	}

	return nil
}

func deserialize(filename string) ([]listItem, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var records []record
	if err := json.NewDecoder(f).Decode(&records); err != nil {
		return nil, err
	}

	var items []listItem
	for _, r := range records {
		var item listItem
		switch r.Type {
		case "github_pull_request":
			item = &pullRequest{
				number: r.Id,
				org:    r.Org,
				repo:   r.Repo,
			}
		case "github_release":
			item = &release{
				tag:  r.Id,
				org:  r.Org,
				repo: r.Repo,
			}
		case "drone_build":
			item = &droneBuild{
				number: r.Id,
				org:    r.Org,
				repo:   r.Repo,
				server: r.Server,
			}
		default:
			return nil, fmt.Errorf("unknown item type: %s", r.Type)
		}
		items = append(items, item)
	}

	return items, nil
}
