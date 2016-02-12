package entity

type Server struct {
	Hostname         string `json:"hostname"`
	IpAddress        string `json:"ipAddress"`
	PrivateIpAddress string `json:"privateIpAddress"`
	IsMaster         bool   `json:"isMaster"`
	IsSlave          bool   `json:"isSlave"`
	IsSwarmMaster    bool   `json:"isSwarmMaster"`
	StoragePath      string `json:"storagePath"`
	IsConsul         bool   `json:"isConsul"`
	IsFullfilled     bool   `json:"isFullfilled"`
	IsDnsServer      bool   `json:"isDnsServer"`
}

type Label struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type DnsConfig struct {
	Zookeeper      string   `json:"zk"`
	Masters        []string `json:"masters"`
	RefreshSeconds int      `json:"refreshSeconds"`
	TimeToLive     int      `json:"ttl"`
	Domain         string   `json:"domain"`
	Port           int      `json:"port"`
	Resolvers      []string `json:"resolvers"`
	Timeout        int      `json:"timeout"`
	HTTPon         bool     `json:"httpon"`
	DNSon          bool     `json:"dnson"`
	HttpPort       int      `json:"httpport"`
	ExternalOn     bool     `json:"externalon"`
	Listener       string   `json:"listener"`
	SOAMname       string   `json:"SOAMname"`
	SOARname       string   `json:"SOARname"`
	SOARefresh     int      `json:"SOARefresh"`
	SOARetry       int      `json:"SOARetry"`
	SOAExpire      int      `json:"SOAExpire"`
	SOAMinttl      int      `json:"SOAMinttl"`
	IPSources      []string `json:"IPSources"`
}

type Parameter struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

type PortMapping struct {
	ContainerPort int    `json:"containerPort"`
	HostPort      int    `json:"hostPort"`
	ServicePort   int    `json:"servicePort"`
	Protocol      string `json:"protocol"`
}

type Docker struct {
	Network        string        `json:"network,omitempty"`
	Image          string        `json:"image,omitempty"`
	Privileged     bool          `json:"privileged,omitempty"`
	ForcePullImage bool          `json:"forcePullImage,omitempty"`
	PortMappings   []PortMapping `json:"portMappings,omitempty"`
	Parameters     []Parameter   `json:"parameters,omitempty"`
}

type Volume struct {
	ContainerPath string `json:"containerPath"`
	HostPath      string `json:"hostPath"`
	Mode          string `json:"mode"`
}

type Container struct {
	Type    string   `json:"type"`
	Docker  Docker   `json:"docker"`
	Volumes []Volume `json:"volumes,omitempty"`
}

type App struct {
	Id          string            `json:"id"`
	Cpus        float32           `json:"cpus"`
	Mem         int16             `json:"mem"`
	Instances   int               `json:"instances"`
	Cmd         string            `json:"cmd,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Container   Container         `json:"container,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Constraints [][]string        `json:"constraints,omitempty"`
	Executor    string            `json:"executor,omitempty"`
}

type Group struct {
	Id           string   `json:"id"`
	Dependencies []string `json:"dependencies,omitempty"`
	Apps         []App    `json:"apps,omitempty"`
	Groups       []Group  `json:"groups,omitempty"`
}

type ServiceGroup struct {
	Id     string  `json:"id"`
	Groups []Group `json:"groups"`
}

type Request struct {
	UserName      string       `json:"username"`
	ClusterName   string       `json:"clusterName"`
	RequestId     string       `json:"requestId"`
	ClusterNumber int          `json:"clusterNumber"`
	IsLinkerMgmt  bool         `json:"isLinkerMgmt"`
	ProviderInfo  ProviderInfo `json:"providerInfo"`
}

type AddNodeRequest struct {
	UserName      string       `json:"username"`
	ClusterName   string       `json:"clusterName"`
	RequestId     string       `json:"requestId"`
	CreateNumber  int          `json:"createNumber"`
	ExistedNumber int          `json:"existedNumber"`
	ConsulServer  string       `json:"consulServer"`
	ProviderInfo  ProviderInfo `json:"providerInfo"`
	DnsServers    []Server     `json:"dnsServers"`
	SwarmMaster   string       `json:"swarmMaster"`
}

type DeleteRequest struct {
	UserName    string   `json:"username"`
	ClusterName string   `json:"clusterName"`
	Servers     []Server `json:"servers"`
}

type ProviderInfo struct {
	Provider      Provider  `json:"provider"`
	OpenstackInfo Openstack `json:"openstackInfo"`
	AwsEC2Info    AwsEC2    `json:"awsEc2Info"`
}

type Provider struct {
	ProviderType string `json:"providerType"`
	SshUser      string `json:"sshUser"`
}

type Openstack struct {
	AuthUrl       string `bson:"authUrl" json:"authUrl"`
	Username      string `bson:"username" json:"username"`
	Password      string `bson:"password" json:"password"`
	TenantName    string `bson:"tenantName" json:"tenantName"`
	FlavorName    string `bson:"flavorName" json:"flavorName"`
	ImageName     string `bson:"imageName" json:"imageName"`
	SecurityGroup string `bson:"securityGroup" json:"securityGroup"`
	IpPoolName    string `bson:"ipPoolName" json:"ipPoolName"`
	NovaNetwork   string `bson:"novaNetwork" json:"novaNetwork"`
}

type AwsEC2 struct {
	AccessKey    string `bson:"accesskeys" json:"accesskeys"`
	SecretKey    string `bson:"secretKey" json:"secretKey"`
	ImageId      string `bson:"imageId" json:"imageId"`
	InstanceType string `bson:"instanceType" json:"instanceType"`
	RootSize     string `bson:"rootSize" json:"rootSize"`
	Region       string `bson:"region" json:"region"`
	VpcId        string `bson:"vpcId" json:"vpcId"`
}
