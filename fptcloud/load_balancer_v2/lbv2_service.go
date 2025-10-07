package fptcloud_load_balancer_v2

import (
	"encoding/json"
	"fmt"
	common "terraform-provider-fptcloud/commons"
)

type LoadBalancerV2Service interface {
	//Load Balancer
	ListLoadBalancers(vpcId string, page int, pageSize int) (LoadBalancerListResponse, error)
	GetLoadBalancer(vpcId string, loadBalancerId string) (LoadBalancerDetailResponse, error)
	CreateLoadBalancer(vpcId string, req LoadBalancerCreateModel) (LoadBalancerResponse, error)
	UpdateLoadBalancer(vpcId string, loadBalancerId string, req LoadBalancerUpdateModel) (LoadBalancerResponse, error)
	ResizeLoadBalancer(vpcId string, loadBalancerId string, req LoadBalancerResizeModel) (LoadBalancerResponse, error)
	DeleteLoadBalancer(vpcId string, loadBalancerId string) (LoadBalancerResponse, error)
	//Listener
	ListListeners(vpcId string, loadBalancerId string, page int, pageSize int) (ListenerListResponse, error)
	GetListener(vpcId string, listenerId string) (ListenerDetailResponse, error)
	CreateListener(vpcId string, loadBalancerId string, req ListenerCreateModel) (ListenerResponse, error)
	UpdateListener(vpcId string, listenerId string, req ListenerUpdateModel) (ListenerResponse, error)
	DeleteListener(vpcId string, listenerId string) (ListenerResponse, error)
	//Pool
	ListPools(vpcId string, loadBalancerId string, page int, pageSize int) (PoolListResponse, error)
	GetPool(vpcId string, poolId string) (PoolDetailResponse, error)
	CreatePool(vpcId string, loadBalancerId string, req PoolCreateModel) (PoolResponse, error)
	UpdatePool(vpcId string, poolId string, req PoolUpdateModel) (PoolResponse, error)
	DeletePool(vpcId string, poolId string) (PoolResponse, error)
	//Certificate
	ListCertificates(vpcId string, page int, pageSize int) (CertificateListResponse, error)
	GetCertificate(vpcId string, certificateId string) (CertificateDetailResponse, error)
	CreateCertificate(vpcId string, req CertificateCreateModel) (CertificateResponse, error)
	DeleteCertificate(vpcId string, certificateId string) (CertificateResponse, error)
	//L7 Policy
	ListL7Policies(vpcId string, listenerId string) (L7PolicyListResponse, error)
	GetL7Policy(vpcId string, listenerId string, policyId string) (L7PolicyDetailResponse, error)
	CreateL7Policy(vpcId string, listenerId string, req L7PolicyInput) (L7PolicyResponse, error)
	UpdateL7Policy(vpcId string, listenerId string, policyId string, req L7PolicyInput) (L7PolicyResponse, error)
	DeleteL7Policy(vpcId string, listenerId string, policyId string) (L7PolicyResponse, error)
	//L7 Rule
	ListL7Rules(vpcId string, listenerId string, policyId string) (L7RuleListResponse, error)
	GetL7Rule(vpcId string, listenerId string, policyId string, ruleId string) (L7RuleDetailResponse, error)
	CreateL7Rule(vpcId string, listenerId string, policyId string, req L7RuleInput) (L7RuleResponse, error)
	UpdateL7Rule(vpcId string, listenerId string, policyId string, ruleId string, req L7RuleInput) (L7RuleResponse, error)
	DeleteL7Rule(vpcId string, listenerId string, policyId string, ruleId string) (L7RuleResponse, error)
	//Size
	ListSizes(vpcId string) (SizeListResponse, error)
}

type LoadBalancerV2ServiceImpl struct {
	client *common.Client
}

func NewLoadBalancerV2Service(client *common.Client) LoadBalancerV2Service {
	return &LoadBalancerV2ServiceImpl{client: client}
}

func (s *LoadBalancerV2ServiceImpl) ListLoadBalancers(vpcId string, page int, pageSize int) (LoadBalancerListResponse, error) {
	apiPath := common.ApiPath.ListLoadBalancers(vpcId, page, pageSize)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return LoadBalancerListResponse{Total: 0}, fmt.Errorf("list load balancers request failed: %v", err)
	}
	var result LoadBalancerListResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return LoadBalancerListResponse{Total: 0}, fmt.Errorf("failed to unmarshal load balancer list response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) GetLoadBalancer(vpcId string, loadBalancerId string) (LoadBalancerDetailResponse, error) {
	apiPath := common.ApiPath.GetLoadBalancer(vpcId, loadBalancerId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return LoadBalancerDetailResponse{}, fmt.Errorf("get load balancer request failed: %v", err)
	}
	var result LoadBalancerDetailResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return LoadBalancerDetailResponse{}, fmt.Errorf("failed to unmarshal load balancer response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) CreateLoadBalancer(vpcId string, req LoadBalancerCreateModel) (LoadBalancerResponse, error) {
	apiPath := common.ApiPath.CreateLoadBalancer(vpcId)
	resp, err := s.client.SendPostRequest(apiPath, req)
	if err != nil {
		return LoadBalancerResponse{}, fmt.Errorf("create load balancer request failed: %v", err)
	}
	var result LoadBalancerResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return LoadBalancerResponse{}, fmt.Errorf("failed to unmarshal load balancer response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) UpdateLoadBalancer(vpcId string, loadBalancerId string, req LoadBalancerUpdateModel) (LoadBalancerResponse, error) {
	apiPath := common.ApiPath.UpdateLoadBalancer(vpcId, loadBalancerId)
	resp, err := s.client.SendPutRequest(apiPath, req)
	if err != nil {
		return LoadBalancerResponse{}, fmt.Errorf("update load balancer request failed: %v", err)
	}
	var result LoadBalancerResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return LoadBalancerResponse{}, fmt.Errorf("failed to unmarshal load balancer response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) ResizeLoadBalancer(vpcId string, loadBalancerId string, req LoadBalancerResizeModel) (LoadBalancerResponse, error) {
	apiPath := common.ApiPath.ResizeLoadBalancer(vpcId, loadBalancerId)
	resp, err := s.client.SendPutRequest(apiPath, req)
	if err != nil {
		return LoadBalancerResponse{}, fmt.Errorf("resize load balancer request failed: %v", err)
	}
	var result LoadBalancerResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return LoadBalancerResponse{}, fmt.Errorf("failed to unmarshal load balancer response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) DeleteLoadBalancer(vpcId string, loadBalancerId string) (LoadBalancerResponse, error) {
	apiPath := common.ApiPath.DeleteLoadBalancer(vpcId, loadBalancerId)
	resp, err := s.client.SendDeleteRequest(apiPath)
	if err != nil {
		return LoadBalancerResponse{}, fmt.Errorf("delete load balancer request failed: %v", err)
	}
	var result LoadBalancerResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return LoadBalancerResponse{}, fmt.Errorf("failed to unmarshal load balancer response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) ListListeners(vpcId string, loadBalancerId string, page int, pageSize int) (ListenerListResponse, error) {
	apiPath := common.ApiPath.ListListeners(vpcId, loadBalancerId, page, pageSize)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return ListenerListResponse{Total: 0}, fmt.Errorf("list listeners request failed: %v", err)
	}
	var result ListenerListResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return ListenerListResponse{Total: 0}, fmt.Errorf("failed to unmarshal list listeners response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) GetListener(vpcId string, listenerId string) (ListenerDetailResponse, error) {
	apiPath := common.ApiPath.GetListener(vpcId, listenerId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return ListenerDetailResponse{}, fmt.Errorf("get listener request fail: %v", err)
	}
	var result ListenerDetailResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return ListenerDetailResponse{}, fmt.Errorf("failed to unmarshal get listener response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) CreateListener(vpcId string, loadBalancerId string, req ListenerCreateModel) (ListenerResponse, error) {
	apiPath := common.ApiPath.CreateListener(vpcId, loadBalancerId)
	resp, err := s.client.SendPostRequest(apiPath, req)
	if err != nil {
		return ListenerResponse{}, fmt.Errorf("create listener request failed: %v", err)
	}
	var result ListenerResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return ListenerResponse{}, fmt.Errorf("failed to unmarshal create listener response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) UpdateListener(vpcId string, listenerId string, req ListenerUpdateModel) (ListenerResponse, error) {
	apiPath := common.ApiPath.UpdateListener(vpcId, listenerId)
	resp, err := s.client.SendPutRequest(apiPath, req)
	if err != nil {
		return ListenerResponse{}, fmt.Errorf("update listener request failed: %v", err)
	}
	var result ListenerResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return ListenerResponse{}, fmt.Errorf("failed to unmarshal update listener response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) DeleteListener(vpcId string, listenerId string) (ListenerResponse, error) {
	apiPath := common.ApiPath.DeleteListener(vpcId, listenerId)
	resp, err := s.client.SendDeleteRequest(apiPath)
	if err != nil {
		return ListenerResponse{}, fmt.Errorf("delete listener request failed: %v", err)
	}
	var result ListenerResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return ListenerResponse{}, fmt.Errorf("failed to unmarshal delete listener response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) ListPools(vpcId string, loadBalancerId string, page int, pageSize int) (PoolListResponse, error) {
	apiPath := common.ApiPath.ListPools(vpcId, loadBalancerId, page, pageSize)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return PoolListResponse{Total: 0}, fmt.Errorf("list pools request fail: %v", err)
	}
	var result PoolListResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return PoolListResponse{Total: 0}, fmt.Errorf("failed to unmarshal list pools response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) GetPool(vpcId string, poolId string) (PoolDetailResponse, error) {
	apiPath := common.ApiPath.GetPool(vpcId, poolId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return PoolDetailResponse{}, fmt.Errorf("get pool request fail: %v", err)
	}
	var result PoolDetailResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return PoolDetailResponse{}, fmt.Errorf("failed to unmarshal get pool response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) CreatePool(vpcId string, loadBalancerId string, req PoolCreateModel) (PoolResponse, error) {
	apiPath := common.ApiPath.CreatePool(vpcId, loadBalancerId)
	resp, err := s.client.SendPostRequest(apiPath, req)
	if err != nil {
		return PoolResponse{}, fmt.Errorf("create pool request fail: %v", err)
	}
	var result PoolResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return PoolResponse{}, fmt.Errorf("failed to unmarshal create pool response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) UpdatePool(vpcId string, poolId string, req PoolUpdateModel) (PoolResponse, error) {
	apiPath := common.ApiPath.UpdatePool(vpcId, poolId)
	resp, err := s.client.SendPutRequest(apiPath, req)
	if err != nil {
		return PoolResponse{}, fmt.Errorf("update pool request fail: %v", err)
	}
	var result PoolResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return PoolResponse{}, fmt.Errorf("failed to unmarshal update pool response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) DeletePool(vpcId string, poolId string) (PoolResponse, error) {
	apiPath := common.ApiPath.DeletePool(vpcId, poolId)
	resp, err := s.client.SendDeleteRequest(apiPath)
	if err != nil {
		return PoolResponse{}, fmt.Errorf("delete pool request fail: %v", err)
	}
	var result PoolResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return PoolResponse{}, fmt.Errorf("failed to unmarshal delete pool response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) ListCertificates(vpcId string, page int, pageSize int) (CertificateListResponse, error) {
	apiPath := common.ApiPath.ListCertificates(vpcId, page, pageSize)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return CertificateListResponse{Total: 0}, fmt.Errorf("list certificates request failed: %v", err)
	}
	var result CertificateListResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return CertificateListResponse{Total: 0}, fmt.Errorf("failed to unmarshal list certificates response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) GetCertificate(vpcId string, certificateId string) (CertificateDetailResponse, error) {
	apiPath := common.ApiPath.GetCertificate(vpcId, certificateId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return CertificateDetailResponse{}, fmt.Errorf("get certificate request failed: %v", err)
	}
	var result CertificateDetailResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return CertificateDetailResponse{}, fmt.Errorf("failed to unmarshal get certificate response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) CreateCertificate(vpcId string, req CertificateCreateModel) (CertificateResponse, error) {
	apiPath := common.ApiPath.CreateCertificate(vpcId)
	resp, err := s.client.SendPostRequest(apiPath, req)
	if err != nil {
		return CertificateResponse{}, fmt.Errorf("create certificate request failed: %v", err)
	}
	var result CertificateResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return CertificateResponse{}, fmt.Errorf("failed to unmarshal create certificate response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) DeleteCertificate(vpcId string, certificateId string) (CertificateResponse, error) {
	apiPath := common.ApiPath.DeleteCertificate(vpcId, certificateId)
	resp, err := s.client.SendDeleteRequest(apiPath)
	if err != nil {
		return CertificateResponse{}, fmt.Errorf("delete certificate request failed: %v", err)
	}
	var result CertificateResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return CertificateResponse{}, fmt.Errorf("failed to unmarshal delete certificate response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) ListL7Policies(vpcId string, listenerId string) (L7PolicyListResponse, error) {
	apiPath := common.ApiPath.ListL7Policies(vpcId, listenerId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return L7PolicyListResponse{Total: 0}, fmt.Errorf("list L7 policies request failed: %v", err)
	}
	var result L7PolicyListResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return L7PolicyListResponse{Total: 0}, fmt.Errorf("failed to unmarshal list L7 policy response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) GetL7Policy(vpcId string, listenerId string, policyId string) (L7PolicyDetailResponse, error) {
	apiPath := common.ApiPath.GetL7Policy(vpcId, listenerId, policyId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return L7PolicyDetailResponse{}, fmt.Errorf("get L7 policy request failed: %v", err)
	}
	var result L7PolicyDetailResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return L7PolicyDetailResponse{}, fmt.Errorf("failed to unmarshal get L7 policy response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) CreateL7Policy(vpcId string, listenerId string, req L7PolicyInput) (L7PolicyResponse, error) {
	apiPath := common.ApiPath.CreateL7Policy(vpcId, listenerId)
	resp, err := s.client.SendPostRequest(apiPath, req)
	if err != nil {
		return L7PolicyResponse{}, fmt.Errorf("create L7 policy request failed: %v", err)
	}
	var result L7PolicyResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return L7PolicyResponse{}, fmt.Errorf("failed to unmarshal create L7 policy response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) UpdateL7Policy(vpcId string, listenerId string, policyId string, req L7PolicyInput) (L7PolicyResponse, error) {
	apiPath := common.ApiPath.UpdateL7Policy(vpcId, listenerId, policyId)
	resp, err := s.client.SendPutRequest(apiPath, req)
	if err != nil {
		return L7PolicyResponse{}, fmt.Errorf("update L7 policy request failed: %v", err)
	}
	var result L7PolicyResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return L7PolicyResponse{}, fmt.Errorf("failed to unmarshal create L7 policy response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) DeleteL7Policy(vpcId string, listenerId string, policyId string) (L7PolicyResponse, error) {
	apiPath := common.ApiPath.DeleteL7Policy(vpcId, listenerId, policyId)
	resp, err := s.client.SendDeleteRequest(apiPath)
	if err != nil {
		return L7PolicyResponse{}, fmt.Errorf("delete L7 policy request failed: %v", err)
	}
	var result L7PolicyResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return L7PolicyResponse{}, fmt.Errorf("failed to unmarshal delete L7 policy response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) ListL7Rules(vpcId string, listenerId string, policyId string) (L7RuleListResponse, error) {
	apiPath := common.ApiPath.ListL7Rules(vpcId, listenerId, policyId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return L7RuleListResponse{Total: 0}, fmt.Errorf("list L7 rules request failed: %v", err)
	}
	var result L7RuleListResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return L7RuleListResponse{Total: 0}, fmt.Errorf("failed to unmarshal list L7 rules response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) GetL7Rule(vpcId string, listenerId string, policyId string, ruleId string) (L7RuleDetailResponse, error) {
	apiPath := common.ApiPath.GetL7Rule(vpcId, listenerId, policyId, ruleId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return L7RuleDetailResponse{}, fmt.Errorf("get L7 rule request failed: %v", err)
	}
	var result L7RuleDetailResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return L7RuleDetailResponse{}, fmt.Errorf("failed to unmarshal get L7 rule response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) CreateL7Rule(vpcId string, listenerId string, policyId string, req L7RuleInput) (L7RuleResponse, error) {
	apiPath := common.ApiPath.CreateL7Rule(vpcId, listenerId, policyId)
	resp, err := s.client.SendPostRequest(apiPath, req)
	if err != nil {
		return L7RuleResponse{}, fmt.Errorf("create L7 rule request failed: %v", err)
	}
	var result L7RuleResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return L7RuleResponse{}, fmt.Errorf("failed to unmarshal create L7 rule response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) UpdateL7Rule(vpcId string, listenerId string, policyId string, ruleId string, req L7RuleInput) (L7RuleResponse, error) {
	apiPath := common.ApiPath.UpdateL7Rule(vpcId, listenerId, policyId, ruleId)
	resp, err := s.client.SendPostRequest(apiPath, req)
	if err != nil {
		return L7RuleResponse{}, fmt.Errorf("update L7 rule request failed: %v", err)
	}
	var result L7RuleResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return L7RuleResponse{}, fmt.Errorf("failed to unmarshal update L7 rule response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) DeleteL7Rule(vpcId string, listenerId string, policyId string, ruleId string) (L7RuleResponse, error) {
	apiPath := common.ApiPath.DeleteL7Rule(vpcId, listenerId, policyId, ruleId)
	resp, err := s.client.SendDeleteRequest(apiPath)
	if err != nil {
		return L7RuleResponse{}, fmt.Errorf("delete L7 rule request failed: %v", err)
	}
	var result L7RuleResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return L7RuleResponse{}, fmt.Errorf("failed to unmarshal delete L7 rule response: %v", err)
	}
	return result, nil
}

func (s *LoadBalancerV2ServiceImpl) ListSizes(vpcId string) (SizeListResponse, error) {
	apiPath := common.ApiPath.ListSizes(vpcId)
	resp, err := s.client.SendGetRequest(apiPath)
	if err != nil {
		return SizeListResponse{Total: 0}, fmt.Errorf("list sizes request failed: %v", err)
	}
	var result SizeListResponse
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return SizeListResponse{Total: 0}, fmt.Errorf("failed to unmarshal list sizes response: %v", err)
	}
	return result, nil
}
