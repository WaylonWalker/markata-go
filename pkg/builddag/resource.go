package builddag

// ResourceKind describes a mutable or shared build resource category.
type ResourceKind string

const (
	ResourceSite     ResourceKind = "site"
	ResourceFiles    ResourceKind = "files"
	ResourcePost     ResourceKind = "post"
	ResourceFeed     ResourceKind = "feed"
	ResourceCache    ResourceKind = "cache"
	ResourceOutput   ResourceKind = "output"
	ResourceTemplate ResourceKind = "template"
	ResourceExternal ResourceKind = "external"
	ResourcePlugin   ResourceKind = "plugin"
)

// AccessMode describes how a task uses a declared resource. A write claim is
// treated as ownership for mutation and therefore also covers any reads the
// task performs while updating that resource.
type AccessMode string

const (
	AccessRead  AccessMode = "read"
	AccessWrite AccessMode = "write"
)

// ResourceID identifies one owned/read build resource. Key is intentionally
// opaque to builddag so callers can use stable post IDs, feed names, cache
// namespaces, output paths, or explicit global sentinels such as "*".
type ResourceID struct {
	Kind ResourceKind `json:"kind"`
	Key  string       `json:"key"`
}

// String returns a deterministic human-readable resource identifier.
func (id ResourceID) String() string {
	return string(id.Kind) + ":" + id.Key
}

// ResourceClaim declares one task's access to a build resource.
type ResourceClaim struct {
	Resource ResourceID `json:"resource"`
	Access   AccessMode `json:"access"`
}

// String returns a deterministic human-readable claim identifier.
func (claim ResourceClaim) String() string {
	return string(claim.Access) + ":" + claim.Resource.String()
}

func resourceClaimLess(a, b ResourceClaim) bool {
	if a.Resource.Kind != b.Resource.Kind {
		return a.Resource.Kind < b.Resource.Kind
	}
	if a.Resource.Key != b.Resource.Key {
		return a.Resource.Key < b.Resource.Key
	}
	return a.Access < b.Access
}

func validResourceClaim(claim ResourceClaim) bool {
	if !validResourceKind(claim.Resource.Kind) || claim.Resource.Key == "" {
		return false
	}
	return claim.Access == AccessRead || claim.Access == AccessWrite
}

func validResourceKind(kind ResourceKind) bool {
	switch kind {
	case ResourceSite, ResourceFiles, ResourcePost, ResourceFeed, ResourceCache,
		ResourceOutput, ResourceTemplate, ResourceExternal, ResourcePlugin:
		return true
	default:
		return false
	}
}
