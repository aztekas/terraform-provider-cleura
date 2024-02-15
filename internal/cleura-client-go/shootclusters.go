package cleura

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (c *Client) GetShootCluster(clusterName string, clusterRegion string, clusterProject string) (*ShootClusterResponse, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/gardener/v1/public/shoot/%s/%s/%s", c.HostURL, clusterRegion, clusterProject, clusterName), nil)
	//https://rest.cleura.cloud/gardener/v1/:gardenDomain/shoot/:region/:project/:shootName
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req, 200, nil)
	if err != nil {
		return nil, err
	}
	shoot := ShootClusterResponse{}
	err = json.Unmarshal(body, &shoot)
	if err != nil {
		return nil, err
	}
	//shoots = append(shoots, shoot)
	return &shoot, nil
}

func (c *Client) CreateShootCluster(clusterRegion string, clusterProject string, shootClusterRequest ShootClusterRequest) (*ShootClusterResponse, error) {
	//https://rest.cleura.cloud/gardener/v1/:gardenDomain/shoot/:region/:project
	crJsonByte, err := json.Marshal(shootClusterRequest)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/gardener/v1/public/shoot/%s/%s", c.HostURL, clusterRegion, clusterProject), strings.NewReader(string(crJsonByte)))
	if err != nil {
		return nil, err
	}
	body, err := c.doRequest(req, 201, nil)
	if err != nil {
		return nil, err
	}
	//fmt.Println("Here's the body!: \n", string(body))
	var createdShootCluster ShootClusterResponse
	err = json.Unmarshal(body, &createdShootCluster)
	if err != nil {
		return nil, err
	}

	return &createdShootCluster, nil
}

func (c *Client) DeleteShootCluster(clusterName string, clusterRegion string, clusterProject string) (string, error) {
	//https://rest.cleura.cloud/gardener/v1/:gardenDomain/shoot/:region/:project/:shoot
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/gardener/v1/public/shoot/%s/%s/%s", c.HostURL, clusterRegion, clusterProject, clusterName), nil)
	if err != nil {
		return "", err
	}
	body, err := c.doRequest(req, 202, nil)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (c *Client) UpdateShootCluster(clusterRegion string, clusterProject string, clusterName string, shootClusterUpdateRequest ShootClusterRequest) (*ShootClusterResponse, error) {
	crJsonByte, err := json.Marshal(shootClusterUpdateRequest)
	if err != nil {
		return nil, err
	}
	//https://rest.cleura.cloud/gardener/v1/:gardenDomain/shoot/:region/:project/:shoot
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/gardener/v1/public/shoot/%s/%s/%s", c.HostURL, clusterRegion, clusterProject, clusterName), strings.NewReader(string(crJsonByte)))
	if err != nil {
		return nil, err
	}
	body, err := c.doRequest(req, 202, nil)
	if err != nil {
		return nil, err
	}
	//fmt.Println("Here's the body!: \n", string(body))
	var createdShootCluster ShootClusterResponse
	err = json.Unmarshal(body, &createdShootCluster)
	fmt.Println(string(body))
	if err != nil {
		return nil, err
	}

	return &createdShootCluster, nil
}
