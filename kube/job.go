package kube

type (
	// RestartPolicyType is a cumstom string type that defines Restart Policy for the
	// Kuberentes job. Currently supported for Jobs are "OnFailure" and "Never"
	RestartPolicyType string

	// ProtocolType is a custom string type defining supported protocols for the pod's
	// exposed ports. Currently supports "udp" and "tcp"
	ProtocolType string
)

const (
	// RestartOnFailure is a Job Restart Policy that restarts the pods on failure
	RestartOnFailure RestartPolicyType = "OnFailure"

	// NeverRestart is a Job Restart Policy that will never restart a failed job
	NeverRestart RestartPolicyType = "Never"

	// UDP is a ProtocolType that defines a "udp" protocol
	UDP ProtocolType = "UDP"

	// TCP is a ProtocolType that defines a "tcp" protocol
	TCP ProtocolType = "TCP"
)

// Job respresents a kubenertes Job
type Job struct {
	APIVersion string                 `json:"apiVersion,omitempty"`
	Kind       string                 `json:"kind,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Spec       *JobSpec               `json:"spec,omitempty"`
	Status     *struct {
		Active     int `json:"active,omitempty"`
		Succeeded  int `json:"succeeded,omitempty"`
		Failed     int `json:"failed,omitempty"`
		Conditions []struct {
			Status string `json:"status,omitempty"`
			Type   string `json:"type,omitempty"`
		} `json:"conditions,omitempty"`
	} `json:"status,omitempty"`
}

// NewJob creates a new Job with default values
func NewJob(name, namespace string) *Job {
	j := &Job{
		APIVersion: "extensions/v1beta1",
		Kind:       "Job",
		Metadata:   make(map[string]interface{}),
		Spec: &JobSpec{
			Selector: &JobSelector{
				MatchLabels: make(map[string]string),
			},
			Template: &JobSpecTemplate{
				Metadata: make(map[string]interface{}),
				Spec: &PodSpec{
					RestartPolicy: NeverRestart,
					Volumes:       make([]*Volume, 0),
					Containers:    make([]*Container, 0),
				},
			},
		},
	}
	j.AddMetadata("name", name)
	j.AddMetadata("namespace", namespace)
	return j
}

// AddMetadata adds a metadata to the job
func (j *Job) AddMetadata(name string, value interface{}) {
	j.Metadata[name] = value
}

// AddLabels add labels to the job
func (j *Job) AddLabels(name, value string) {
	if _, ok := j.Metadata["labels"]; !ok {
		j.Metadata["labels"] = make(map[string]string)
	}
	j.Metadata["labels"].(map[string]string)[name] = value
}

// AddAnnotations adds annotations to the job
func (j *Job) AddAnnotations(name, value string) {
	if _, ok := j.Metadata["annotations"]; !ok {
		j.Metadata["annotations"] = make(map[string]string)
	}
	j.Metadata["annotations"].(map[string]string)[name] = value
}

// AddSelectorMatchLabel adds key=value labels to the selector and the job metadata
func (j *Job) AddSelectorMatchLabel(name, value string) {
	selector := j.Spec.Selector.MatchLabels
	metadata := j.Spec.Template.Metadata
	if _, ok := metadata["labels"]; !ok {
		metadata["labels"] = make(map[string]string)
	}
	selector[name] = value
	metadata["labels"].(map[string]string)[name] = value
}

// AddPodVolume adds a new volume to the pod. Reference to the created volume is returned
func (j *Job) AddPodVolume(name, path string) *Volume {
	vol := &Volume{
		Name:     name,
		HostPath: &HostPathVolume{path},
	}
	j.Spec.Template.Spec.Volumes = append(j.Spec.Template.Spec.Volumes, vol)
	return vol
}

// AddPodContainer adds a new container to the pod. Reference to the created pod is returned
func (j *Job) AddPodContainer(name, image string) *Container {
	container := &Container{
		Name:            name,
		Image:           image,
		ImagePullPolicy: "Always",
	}
	j.Spec.Template.Spec.Containers = append(j.Spec.Template.Spec.Containers, container)
	return container
}

// JobSpec defines the job selector and template
type JobSpec struct {
	Selector *JobSelector     `json:"selector,omitempty"`
	Template *JobSpecTemplate `json:"template,omitempty"`
}

// JobSelector defines a map of labels to select the pods for the job
type JobSelector struct {
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

// JobSpecTemplate defines the metadata and pod template
type JobSpecTemplate struct {
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Spec     *PodSpec               `json:"spec,omitempty"`
}

// PodSpec defines the specs of the pod
type PodSpec struct {
	Volumes       []*Volume         `json:"volumes,omitempty"`
	Containers    []*Container      `json:"containers,omitempty"`
	RestartPolicy RestartPolicyType `json:"restartPolicy,omitempty"`
}

// Volume defines a kubernetes volume for the pod
type Volume struct {
	Name     string          `json:"name,omitempty"`
	HostPath *HostPathVolume `json:"hostPath,omitempty"`
	EmptyDir *EmptyDirVolume `json:"emptyDir,omitempty"`
	// add more volumes?
}

// HostPathVolume is a volume that is located on the host path
type HostPathVolume struct {
	Path string `json:"path,omitempty"`
}

// EmptyDirVolume is a empty directory used as a volume
type EmptyDirVolume struct {
	Medium string `json:"medium,omitempty"`
}

// Container defines a container in a kubernetes pod
type Container struct {
	Name            string           `json:"name,omitempty"`
	Image           string           `json:"image,omitempty"`
	ImagePullPolicy string           `json:"imagePullPolicy,omitempty"`
	WorkingDir      string           `json:"workingDir,omitempty"`
	Command         []string         `json:"command,omitempty"`
	Args            []string         `json:"args,omitempty"`
	Ports           []*ContainerPort `json:"ports,omitempty"`
	Env             []*EnvVar        `json:"env,omitempty"`
	VolumeMounts    []*VolumeMount   `json:"volumeMounts,omitempty"`
}

// SetCommand sets the command for the container
func (c *Container) SetCommand(command ...string) {
	c.Command = command
}

// SetArgs sets the arguments for the container
func (c *Container) SetArgs(args ...string) {
	c.Args = args
}

// AddPort adds a new port to the container
func (c *Container) AddPort(name string, port int, protocol ProtocolType) {
	if c.Ports == nil {
		c.Ports = make([]*ContainerPort, 0)
	}
	p := &ContainerPort{
		Name:          name,
		ContainerPort: port,
		Protocol:      protocol,
	}
	c.Ports = append(c.Ports, p)
}

// AddEnv adds a new environment variable to the container
func (c *Container) AddEnv(name, value string) {
	if c.Env == nil {
		c.Env = make([]*EnvVar, 0)
	}
	c.Env = append(c.Env, &EnvVar{name, value})
}

// AddVolumeMountPoint adds a mount point to the container for the given volume
func (c *Container) AddVolumeMountPoint(volume *Volume, mountPath string, readOnly bool) {
	if c.VolumeMounts == nil {
		c.VolumeMounts = make([]*VolumeMount, 0)
	}
	vm := &VolumeMount{
		Name:      volume.Name,
		MountPath: mountPath,
		ReadOnly:  readOnly,
	}
	c.VolumeMounts = append(c.VolumeMounts, vm)
}

// ContainerPort defines a port exposed by a container
type ContainerPort struct {
	Name          string       `json:"name,omitempty"`
	ContainerPort int          `json:"containerPort,omitempty"`
	Protocol      ProtocolType `json:"protocol,omitempty"`
	// not adding hostPort and hostIP
}

// EnvVar defines an environment variable for the container
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// VolumeMount defines where a volume will be mounted in a container
type VolumeMount struct {
	Name      string `json:"name,omitempty"`
	MountPath string `json:"mountPath,omitempty"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
}
