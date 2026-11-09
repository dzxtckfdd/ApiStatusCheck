package client

import "apiChecker/internal/dto"

type CatalogClient struct {
}

func (cc *CatalogClient) GetActiveResources() (*dto.ResourcesResponce, error) {

}

func (cc *CatalogClient) SendBatchChecks(batch *dto.SaveBatchResponce) error {
	
}
