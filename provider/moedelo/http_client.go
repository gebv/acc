package moedelo

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

type client struct {
	httpClient *http.Client
	token      string
}

func newClient(token string) *client {
	return &client{
		httpClient: &http.Client{},
		token:      token,
	}
}

func (c *client) GETAndUnmarshalJson(link string, out interface{}) error {
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return errors.Wrap(err, "Failed new request")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("md-api-key", c.token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "Failed do request")
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "Failed read all body")
	}
	err = json.Unmarshal(b, out)
	if err != nil {
		return errors.Wrap(err, "Failed unmarshal")
	}
	if resp.StatusCode == 404 {
		return errors.New("not_found")
	}
	if resp.StatusCode == 422 {
		return errors.New("error_validation")
	}
	return nil

}

func (c *client) DELETE(link string) error {
	req, err := http.NewRequest("DELETE", link, nil)
	if err != nil {
		return errors.Wrap(err, "Failed new request")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("md-api-key", c.token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "Failed do request")
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return errors.New("not_found")
	}
	return nil
}

func (c *client) POSTAndUnmarshalJson(link string, in, out interface{}) error {
	b, err := json.Marshal(in)
	if err != nil {
		return errors.Wrap(err, "Failed marshal")
	}
	req, err := http.NewRequest("POST", link, bytes.NewReader(b))
	if err != nil {
		return errors.Wrap(err, "Failed new request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("md-api-key", c.token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "Failed do request")
	}
	defer resp.Body.Close()
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "Failed read all body")
	}
	err = json.Unmarshal(b, out)
	if err != nil {
		return errors.Wrap(err, "Failed unmarshal")
	}
	if resp.StatusCode == 404 {
		return errors.New("not_found")
	}
	if resp.StatusCode == 422 {
		return errors.New("error_validation")
	}
	return nil
}

func (c *client) PUTAndUnmarshalJson(link string, in, out interface{}) error {
	b, err := json.Marshal(in)
	if err != nil {
		return errors.Wrap(err, "Failed marshal")
	}
	req, err := http.NewRequest("PUT", link, bytes.NewReader(b))
	if err != nil {
		return errors.Wrap(err, "Failed new request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("md-api-key", c.token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "Failed do request")
	}
	defer resp.Body.Close()
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "Failed read all body")
	}
	err = json.Unmarshal(b, out)
	if err != nil {
		return errors.Wrap(err, "Failed unmarshal")
	}
	if resp.StatusCode == 404 {
		return errors.New("not_found")
	}
	if resp.StatusCode == 422 {
		return errors.New("error_validation")
	}
	return nil
}
