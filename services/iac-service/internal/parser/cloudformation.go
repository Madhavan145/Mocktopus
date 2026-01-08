package parser

import (
	"encoding/json"
	"fmt"
	"strings"

	"moktopus/iac-service/internal/model"
)

// ParseCloudFormation parses AWS CloudFormation JSON/YAML into a CTM graph.
func ParseCloudFormation(content string) (*model.ImportResponse, error) {
	resp := &model.ImportResponse{
		Format:   "cloudformation",
		Provider: "aws",
		Warnings: []string{},
	}

	var template cfnTemplate
	if err := json.Unmarshal([]byte(content), &template); err != nil {
		template = parseSimpleYAML(content)
	}

	// Detect region from parameters/mappings
	if region := extractCFNRegion(content); region != "" {
		resp.Region = region
	}

	nodes := []model.Node{}
	edges := []model.Edge{}
	nodeIDs := map[string]bool{}
	edgeSet := map[string]bool{}

	for logicalID, resource := range template.Resources {
		nodeType, config := mapCFNResource(resource.Type, logicalID, resource.Properties)
		if nodeType == "" {
			resp.Warnings = append(resp.Warnings, fmt.Sprintf("unsupported: %s (%s)", logicalID, resource.Type))
			continue
		}

		nodes = append(nodes, model.Node{
			ID:     logicalID,
			Type:   nodeType,
			Config: config,
		})
		nodeIDs[logicalID] = true
	}

	// Detect edges from Ref, GetAtt, DependsOn, and Sub
	for logicalID, resource := range template.Resources {
		if !nodeIDs[logicalID] {
			continue
		}
		// Property references
		refs := extractRefs(resource.Properties)
		for _, ref := range refs {
			if nodeIDs[ref] && ref != logicalID {
				edgeKey := logicalID + "->" + ref
				if !edgeSet[edgeKey] {
					edgeSet[edgeKey] = true
					edges = append(edges, model.Edge{
						ID:     fmt.Sprintf("ref-%s-%s", logicalID, ref),
						Source: logicalID,
						Target: ref,
					})
				}
			}
		}
		// DependsOn
		for _, dep := range resource.DependsOn {
			if nodeIDs[dep] && dep != logicalID {
				edgeKey := logicalID + "->" + dep
				if !edgeSet[edgeKey] {
					edgeSet[edgeKey] = true
					edges = append(edges, model.Edge{
						ID:     fmt.Sprintf("dep-%s-%s", logicalID, dep),
						Source: logicalID,
						Target: dep,
					})
				}
			}
		}
	}

	resp.Graph = model.Graph{Nodes: nodes, Edges: edges}
	resp.ResourceCount = len(nodes)
	return resp, nil
}

type cfnTemplate struct {
	Resources map[string]cfnResource `json:"Resources"`
}

type cfnResource struct {
	Type       string         `json:"Type"`
	Properties map[string]any `json:"Properties"`
	DependsOn  []string       `json:"DependsOn"`
}

func extractCFNRegion(content string) string {
	// Look for region in parameters or default region
	if idx := strings.Index(content, "us-east-1"); idx >= 0 {
		return "us-east-1"
	}
	if idx := strings.Index(content, "us-west-2"); idx >= 0 {
		return "us-west-2"
	}
	if idx := strings.Index(content, "eu-west-1"); idx >= 0 {
		return "eu-west-1"
	}
	return ""
}

func parseSimpleYAML(content string) cfnTemplate {
	tmpl := cfnTemplate{Resources: map[string]cfnResource{}}
	lines := strings.Split(content, "\n")
	inResources := false
	currentResource := ""
	currentType := ""
	var currentProps map[string]any
	inProps := false
	currentPropKey := ""
	inDependsOn := false
	var dependsOn []string

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if trimmed == "Resources:" {
			inResources = true
			continue
		}

		// Exit Resources section if we hit another top-level key
		if inResources && len(line) > 0 && line[0] != ' ' && strings.HasSuffix(trimmed, ":") && trimmed != "Resources:" {
			inResources = false
			if currentResource != "" && currentType != "" {
				tmpl.Resources[currentResource] = cfnResource{Type: currentType, Properties: currentProps, DependsOn: dependsOn}
			}
			continue
		}

		if !inResources {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))

		// Resource name (2-space indent)
		if indent == 2 && strings.HasSuffix(trimmed, ":") && !strings.Contains(trimmed, " ") {
			// Save previous resource
			if currentResource != "" && currentType != "" {
				tmpl.Resources[currentResource] = cfnResource{Type: currentType, Properties: currentProps, DependsOn: dependsOn}
			}
			currentResource = strings.TrimSuffix(trimmed, ":")
			currentType = ""
			currentProps = map[string]any{}
			inProps = false
			inDependsOn = false
			dependsOn = nil
			continue
		}

		// Type field (4-space indent)
		if indent == 4 && strings.HasPrefix(trimmed, "Type:") {
			currentType = strings.TrimSpace(strings.TrimPrefix(trimmed, "Type:"))
			continue
		}

		// Properties section (4-space indent)
		if indent == 4 && trimmed == "Properties:" {
			inProps = true
			inDependsOn = false
			continue
		}

		// DependsOn section (4-space indent)
		if indent == 4 && trimmed == "DependsOn:" {
			inDependsOn = true
			inProps = false
			continue
		}

		if inDependsOn && indent >= 6 {
			if strings.HasPrefix(trimmed, "- ") {
				dep := strings.TrimPrefix(trimmed, "- ")
				dependsOn = append(dependsOn, strings.TrimSpace(dep))
			}
			continue
		}

		if indent == 4 && !strings.HasPrefix(trimmed, "Type:") && strings.HasSuffix(trimmed, ":") {
			inDependsOn = false
			inProps = false
		}

		// Property key-value (6-space indent)
		if inProps && indent == 6 {
			if strings.Contains(trimmed, ":") && !strings.HasSuffix(trimmed, ":") {
				parts := strings.SplitN(trimmed, ":", 2)
				key := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				currentProps[key] = val
				currentPropKey = key
			} else if strings.HasSuffix(trimmed, ":") {
				currentPropKey = strings.TrimSuffix(trimmed, ":")
			}
			continue
		}

		// Nested property value (8+ space indent)
		if inProps && indent >= 8 && currentPropKey != "" {
			if strings.HasPrefix(trimmed, "- !Ref ") {
				ref := strings.TrimPrefix(trimmed, "- !Ref ")
				existing, _ := currentProps[currentPropKey].([]any)
				currentProps[currentPropKey] = append(existing, map[string]any{"Ref": strings.TrimSpace(ref)})
			} else if strings.HasPrefix(trimmed, "!Ref ") {
				ref := strings.TrimPrefix(trimmed, "!Ref ")
				currentProps[currentPropKey] = map[string]any{"Ref": strings.TrimSpace(ref)}
			}
		}

		_ = currentPropKey
	}

	// Save last resource
	if currentResource != "" && currentType != "" {
		tmpl.Resources[currentResource] = cfnResource{Type: currentType, Properties: currentProps, DependsOn: dependsOn}
	}

	return tmpl
}

func mapCFNResource(cfnType, logicalID string, props map[string]any) (string, map[string]any) {
	config := map[string]any{
		"label":       logicalID,
		"iacResource": cfnType,
		"provider":    "aws",
	}

	switch {
	// ═══ NETWORKING ═══
	case cfnType == "AWS::EC2::VPC":
		if v, ok := props["CidrBlock"]; ok {
			config["cidrBlock"] = v
		}
		return "vpc", config
	case cfnType == "AWS::EC2::Subnet":
		if v, ok := props["CidrBlock"]; ok {
			config["cidrBlock"] = v
		}
		if v, ok := props["AvailabilityZone"]; ok {
			config["availabilityZone"] = v
		}
		if v, ok := props["MapPublicIpOnLaunch"]; ok {
			config["public"] = v
		}
		return "subnet", config
	case cfnType == "AWS::EC2::InternetGateway":
		return "internet-gateway", config
	case cfnType == "AWS::EC2::NatGateway":
		return "nat-gateway", config
	case cfnType == "AWS::EC2::RouteTable" || cfnType == "AWS::EC2::Route" || cfnType == "AWS::EC2::SubnetRouteTableAssociation":
		return "route-table", config
	case cfnType == "AWS::EC2::SecurityGroup":
		if v, ok := props["GroupDescription"]; ok {
			config["description"] = v
		}
		return "security-group", config
	case cfnType == "AWS::EC2::EIP":
		return "elastic-ip", config
	case cfnType == "AWS::EC2::VPCGatewayAttachment":
		return "internet-gateway", config

	// ═══ COMPUTE ═══
	case cfnType == "AWS::EC2::Instance":
		if v, ok := props["InstanceType"]; ok {
			config["instanceType"] = v
		}
		if v, ok := props["ImageId"]; ok {
			config["ami"] = v
		}
		return "service", config
	case cfnType == "AWS::ECS::Cluster":
		return "service", config
	case cfnType == "AWS::ECS::Service":
		if v, ok := props["DesiredCount"]; ok {
			config["desiredCount"] = v
		}
		if v, ok := props["LaunchType"]; ok {
			config["launchType"] = v
		}
		if v, ok := props["ServiceName"]; ok {
			config["serviceName"] = v
		}
		return "service", config
	case cfnType == "AWS::ECS::TaskDefinition":
		if v, ok := props["Cpu"]; ok {
			config["cpu"] = v
		}
		if v, ok := props["Memory"]; ok {
			config["memory"] = v
		}
		return "service", config
	case cfnType == "AWS::AutoScaling::AutoScalingGroup":
		if v, ok := props["MinSize"]; ok {
			config["minSize"] = v
		}
		if v, ok := props["MaxSize"]; ok {
			config["maxSize"] = v
		}
		if v, ok := props["DesiredCapacity"]; ok {
			config["desiredCapacity"] = v
		}
		return "service", config
	case cfnType == "AWS::EC2::LaunchTemplate":
		return "service", config
	case cfnType == "AWS::AppRunner::Service":
		return "service", config

	// ═══ SERVERLESS ═══
	case cfnType == "AWS::Lambda::Function":
		config["runtime"] = "serverless"
		if v, ok := props["Runtime"]; ok {
			config["lambdaRuntime"] = v
		}
		if v, ok := props["MemorySize"]; ok {
			config["memorySize"] = v
		}
		if v, ok := props["Timeout"]; ok {
			config["timeout"] = v
		}
		return "serverless", config
	case cfnType == "AWS::Lambda::EventSourceMapping":
		return "serverless", config
	case cfnType == "AWS::Serverless::Function":
		config["runtime"] = "serverless"
		return "serverless", config

	// ═══ DATABASE ═══
	case cfnType == "AWS::RDS::DBInstance":
		if v, ok := props["Engine"]; ok {
			config["engine"] = v
		}
		if v, ok := props["EngineVersion"]; ok {
			config["engineVersion"] = v
		}
		if v, ok := props["DBInstanceClass"]; ok {
			config["instanceClass"] = v
		}
		if v, ok := props["MultiAZ"]; ok {
			config["multiAz"] = v
		}
		if v, ok := props["StorageType"]; ok {
			config["storageType"] = v
		}
		if v, ok := props["AllocatedStorage"]; ok {
			config["allocatedStorage"] = v
		}
		return "database", config
	case cfnType == "AWS::RDS::DBCluster":
		if v, ok := props["Engine"]; ok {
			config["engine"] = v
		}
		if v, ok := props["EngineVersion"]; ok {
			config["engineVersion"] = v
		}
		return "database", config
	case cfnType == "AWS::DynamoDB::Table":
		if v, ok := props["BillingMode"]; ok {
			config["billingMode"] = v
		}
		return "database", config

	// ═══ CACHE ═══
	case cfnType == "AWS::ElastiCache::ReplicationGroup":
		if v, ok := props["CacheNodeType"]; ok {
			config["nodeType"] = v
		}
		if v, ok := props["Engine"]; ok {
			config["engine"] = v
		}
		if v, ok := props["NumCacheClusters"]; ok {
			config["numClusters"] = v
		}
		return "cache", config
	case cfnType == "AWS::ElastiCache::CacheCluster":
		if v, ok := props["CacheNodeType"]; ok {
			config["nodeType"] = v
		}
		return "cache", config

	// ═══ QUEUE / MESSAGING ═══
	case cfnType == "AWS::SQS::Queue":
		if v, ok := props["QueueName"]; ok {
			config["queueName"] = v
		}
		if v, ok := props["VisibilityTimeout"]; ok {
			config["visibilityTimeout"] = v
		}
		if v, ok := props["FifoQueue"]; ok {
			config["fifo"] = v
		}
		return "queue", config
	case cfnType == "AWS::SNS::Topic":
		if v, ok := props["TopicName"]; ok {
			config["topicName"] = v
		}
		return "notification", config
	case cfnType == "AWS::SNS::Subscription":
		if v, ok := props["Protocol"]; ok {
			config["protocol"] = v
		}
		return "notification", config
	case cfnType == "AWS::Kinesis::Stream":
		if v, ok := props["ShardCount"]; ok {
			config["shardCount"] = v
		}
		return "stream", config
	case cfnType == "AWS::MSK::Cluster":
		return "stream", config
	case cfnType == "AWS::Events::EventBus" || cfnType == "AWS::Events::Rule":
		return "stream", config

	// ═══ LOAD BALANCER ═══
	case cfnType == "AWS::ElasticLoadBalancingV2::LoadBalancer":
		if v, ok := props["Type"]; ok {
			config["lbType"] = v
		}
		if v, ok := props["Scheme"]; ok {
			config["scheme"] = v
		}
		return "load-balancer", config
	case cfnType == "AWS::ElasticLoadBalancingV2::TargetGroup":
		if v, ok := props["Port"]; ok {
			config["port"] = v
		}
		if v, ok := props["Protocol"]; ok {
			config["protocol"] = v
		}
		return "load-balancer", config
	case cfnType == "AWS::ElasticLoadBalancingV2::Listener":
		if v, ok := props["Port"]; ok {
			config["port"] = v
		}
		return "load-balancer", config
	case cfnType == "AWS::ElasticLoadBalancing::LoadBalancer":
		return "load-balancer", config

	// ═══ API GATEWAY ═══
	case cfnType == "AWS::ApiGateway::RestApi" || cfnType == "AWS::ApiGatewayV2::Api":
		if v, ok := props["Name"]; ok {
			config["apiName"] = v
		}
		return "api-gateway", config
	case cfnType == "AWS::ApiGateway::Stage" || cfnType == "AWS::ApiGatewayV2::Stage":
		return "api-gateway", config

	// ═══ CDN ═══
	case cfnType == "AWS::CloudFront::Distribution":
		return "cdn", config

	// ═══ STORAGE ═══
	case cfnType == "AWS::S3::Bucket":
		if v, ok := props["BucketName"]; ok {
			config["bucketName"] = v
		}
		return "storage", config
	case cfnType == "AWS::EFS::FileSystem":
		return "storage", config

	// ═══ DNS ═══
	case cfnType == "AWS::Route53::HostedZone" || cfnType == "AWS::Route53::RecordSet":
		return "dns", config

	// ═══ FIREWALL / WAF ═══
	case cfnType == "AWS::WAFv2::WebACL" || cfnType == "AWS::WAF::WebACL":
		return "firewall", config

	// ═══ CONTAINER / K8S ═══
	case cfnType == "AWS::EKS::Cluster":
		return "kubernetes", config
	case cfnType == "AWS::EKS::Nodegroup":
		if v, ok := props["InstanceTypes"]; ok {
			config["instanceTypes"] = v
		}
		return "kubernetes", config

	// ═══ VPN ═══
	case cfnType == "AWS::EC2::VPNGateway" || cfnType == "AWS::EC2::VPNConnection":
		return "vpn", config

	// ═══ MONITORING ═══
	case cfnType == "AWS::CloudWatch::Alarm" || cfnType == "AWS::Logs::LogGroup":
		return "monitoring", config

	// ═══ IAM ═══
	case cfnType == "AWS::IAM::Role" || cfnType == "AWS::IAM::Policy" || cfnType == "AWS::IAM::InstanceProfile":
		return "iam", config

	default:
		// Try to infer from type prefix
		if strings.Contains(cfnType, "EC2") || strings.Contains(cfnType, "ECS") || strings.Contains(cfnType, "AppRunner") {
			return "service", config
		}
		return "service", config
	}
}

func extractRefs(props map[string]any) []string {
	refs := []string{}
	for _, v := range props {
		switch val := v.(type) {
		case string:
			// Check for !Ref shorthand in YAML-parsed values
			if strings.HasPrefix(val, "!Ref ") {
				refs = append(refs, strings.TrimPrefix(val, "!Ref "))
			}
		case map[string]any:
			if ref, ok := val["Ref"]; ok {
				if s, ok := ref.(string); ok && !strings.HasPrefix(s, "AWS::") {
					refs = append(refs, s)
				}
			}
			if getAtt, ok := val["Fn::GetAtt"]; ok {
				if arr, ok := getAtt.([]any); ok && len(arr) > 0 {
					if s, ok := arr[0].(string); ok {
						refs = append(refs, s)
					}
				}
				// String form: "Resource.Attribute"
				if s, ok := getAtt.(string); ok {
					parts := strings.Split(s, ".")
					if len(parts) > 0 {
						refs = append(refs, parts[0])
					}
				}
			}
			if sub, ok := val["Fn::Sub"]; ok {
				if s, ok := sub.(string); ok {
					// Extract ${Resource} or ${Resource.Attr} patterns
					for _, part := range strings.Split(s, "${") {
						if idx := strings.Index(part, "}"); idx > 0 {
							ref := part[:idx]
							ref = strings.Split(ref, ".")[0]
							if !strings.HasPrefix(ref, "AWS::") && ref != "" {
								refs = append(refs, ref)
							}
						}
					}
				}
			}
			refs = append(refs, extractRefs(val)...)
		case []any:
			for _, item := range val {
				if m, ok := item.(map[string]any); ok {
					refs = append(refs, extractRefs(m)...)
				}
				if s, ok := item.(string); ok {
					if strings.HasPrefix(s, "!Ref ") {
						refs = append(refs, strings.TrimPrefix(s, "!Ref "))
					}
				}
			}
		}
	}
	return refs
}
