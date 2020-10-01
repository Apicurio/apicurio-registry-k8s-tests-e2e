package resources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	Avro ArtifactType = "AVRO"
)

type ArtifactType string

type ApicurioRegistryApiClient interface {
	CreateArtifact(id string, artifactType ArtifactType, data string) error
	ReadArtifact(id string) (string, error)
	DeleteArtifact(id string) error
	ListArtifacts() ([]string, error)
}

type ApicurioRegistryApiClientImpl struct {
	host       string
	port       string
	httpClient *http.Client
}

func NewApicurioRegistryApiClient(host string, port string, httpClient *http.Client) ApicurioRegistryApiClient {
	return &ApicurioRegistryApiClientImpl{
		host:       host,
		port:       port,
		httpClient: httpClient,
	}
}

func (r *ApicurioRegistryApiClientImpl) CreateArtifact(id string, artifactType ArtifactType, data string) error {
	url := fmt.Sprintf("http://%v:%v/api/artifacts", r.host, r.port)
	body := bytes.NewBuffer([]byte(data))

	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return err
	}

	switch artifactType {
	case Avro:
		req.Header.Set("Content-Type", fmt.Sprintf("application/json; artifactType=%v", Avro))
	}

	req.Header.Set("X-Registry-ArtifactId", id)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(fmt.Sprintf("expected status 200 but received %v", resp.StatusCode))
	}

	return nil
}

func (r *ApicurioRegistryApiClientImpl) ReadArtifact(id string) (string, error) {
	url := fmt.Sprintf("http://%v:%v/api/artifacts", r.host, r.port)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(fmt.Sprintf("expected status 200 but received %v", resp.StatusCode))
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	data := string(bytes)

	return data, nil
}

func (r *ApicurioRegistryApiClientImpl) DeleteArtifact(id string) error {
	url := fmt.Sprintf("http://%v:%v/api/artifacts", r.host, r.port)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf(fmt.Sprintf("expected status %v but received %v", http.StatusNoContent, resp.StatusCode))
	}

	return nil
}

func (r *ApicurioRegistryApiClientImpl) ListArtifacts() ([]string, error) {
	url := fmt.Sprintf("http://%v:%v/api/artifacts", r.host, r.port)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(fmt.Sprintf("expected status 200 but received %v", resp.StatusCode))
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var list []string = make([]string, 0)
	err = json.Unmarshal(bytes, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}
