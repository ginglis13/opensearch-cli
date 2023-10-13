/*
 * SPDX-License-Identifier: Apache-2.0
 *
 * The OpenSearch Contributors require contributions made to
 * this file be licensed under the Apache-2.0 license or a
 * compatible open source license.
 *
 * Modifications Copyright OpenSearch Contributors. See
 * GitHub history for details.
 */

package it

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"opensearch-cli/client"
	"opensearch-cli/entity"
	"opensearch-cli/environment"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/suite"
)

const (
	newLine           = "\n"
	getPluginNamesURL = "_cat/plugins?h=c"
)

type CLISuite struct {
	suite.Suite
	Client  *client.Client
	Profile *entity.Profile
	Plugins []string
}

// HelperLoadBytes loads file from testdata and stream contents
func HelperLoadBytes(name string) []byte {
	path := filepath.Join("testdata", name) // relative path
	contents, err := os.ReadFile(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return contents
}

// DeleteIndex deletes index by name
func (a *CLISuite) DeleteIndex(indexName string) {
	_, err := a.callRequest(http.MethodDelete, []byte(""), fmt.Sprintf("%s/%s", a.Profile.Endpoint, indexName))

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func (a *CLISuite) ValidateProfile() error {
	if a.Profile.Endpoint == "" {
		return fmt.Errorf("endpoint cannot be empty. set env %s", environment.OPENSEARCH_ENDPOINT)
	}
	if len(a.Profile.UserName) == 0 {
		return nil
	}
	if a.Profile.Password == "" {
		return fmt.Errorf("password cannot be empty. set env %s", environment.OPENSEARCH_PASSWORD)
	}
	return nil
}

// CreateIndex creates test data for plugin processing
func (a *CLISuite) CreateIndex(indexFileName string, mappingFileName string) {
	if mappingFileName != "" {
		mapping, err := a.callRequest(
			http.MethodPut, HelperLoadBytes(mappingFileName), fmt.Sprintf("%s/%s", a.Profile.Endpoint, indexFileName))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println(string(mapping))
	}
	res, err := a.callRequest(
		http.MethodPost, HelperLoadBytes(indexFileName), fmt.Sprintf("%s/_bulk?refresh", a.Profile.Endpoint))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(string(res))
}

func (a *CLISuite) callRequest(method string, reqBytes []byte, url string) ([]byte, error) {
	var reqReader *bytes.Reader
	if reqBytes != nil {
		reqReader = bytes.NewReader(reqBytes)
	}
	r, err := retryablehttp.NewRequest(method, url, reqReader)
	if err != nil {
		return nil, err
	}
	req := r.WithContext(context.Background())
	req.SetBasicAuth(a.Profile.UserName, a.Profile.Password)
	req.Header.Set("Content-Type", "application/x-ndjson")
	response, err := a.Client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := response.Body.Close()
		if err != nil {
			return
		}
	}()
	return io.ReadAll(response.Body)
}

// isPluginInstalled checks whether dependent plugins are insalled or not
func (a *CLISuite) IsPluginInstalled() bool {
	return a.IsPluginFromInputInstalled(a.Plugins)
}

// IsPluginFromInputInstalled checks whether input plugins are insalled or not
func (a *CLISuite) IsPluginFromInputInstalled(plugins []string) bool {
	pluginListsInBytes, err := a.callRequest(http.MethodGet, []byte(""), fmt.Sprintf("%s/%s", a.Profile.Endpoint, getPluginNamesURL))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	pluginListsAsString := string(pluginListsInBytes[:])
	pluginArray := strings.Split(pluginListsAsString, newLine)
	for _, plugin := range plugins {
		if !contains(pluginArray, plugin) {
			return false
		}
	}
	return true
}

func contains(container []string, value string) bool {
	for _, item := range container {
		if item == value {
			return true
		}
	}
	return false
}
