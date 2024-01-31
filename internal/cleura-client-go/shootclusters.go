package cleura

import (
	"encoding/json"
	"fmt"
	"net/http"
)

//Mockup
// Fields you fill in when creating via web console

// type TestShootCluster struct {
//- 	Name                 string
//- 	Region               string
//- 	KubernetesVersion    string
// 	ExternalNetwork      string //hardcoded to ext-net
// 	WorkerGroups         []WorkerGroup
// 	MaintenanceWindow    MaintenanceWindowDetails
// 	HibernationSchedules []HibernationSchedule
// 	AdvancedSettings     AdvancedSetting
// }

// type WorkerGroup struct {
// 	Name          string
// 	Flavour       string // choice list
// 	ImageOS       string // hardcoded to gardenlinux
// 	VolumeSize    string //or int? in GB
// 	AutoscalerMin int
// 	AutoscalerMax int
// 	MaxSurge      int
// }

// type MaintenanceWindowDetails struct {
// 	Time     string
// 	TimeZone string
// }

// type HibernationSchedule struct {
// }

// type AdvancedSetting struct {
// 	WorkersCIDR string
// }

///

// type ShootCluster struct{

// }

func (c *Client) GetShootCluster(clusterName string, clusterRegion string, clusterProject string) (*ShootClusterResponse, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/gardener/v1/public/shoot/%s/%s/%s", c.HostURL, clusterRegion, clusterProject, clusterName), nil)
	//https://rest.cleura.cloud/gardener/v1/:gardenDomain/shoot/:region/:project/:shootName
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req, nil)
	if err != nil {
		return nil, err
	}
	shoot := ShootClusterResponse{}
	//shoots := []ShootCluster{}
	err = json.Unmarshal(body, &shoot)
	if err != nil {
		return nil, err
	}
	//shoots = append(shoots, shoot)
	return &shoot, nil
}
