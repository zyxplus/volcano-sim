package api

// Resource is a quantity vector keyed by resource name, for example cpu, gpu,
// or an extended resource such as rdma.
type Resource map[string]float64

// NewResource copies values so callers cannot mutate the returned resource via
// the source map.
func NewResource(values map[string]float64) Resource {
	resource := make(Resource, len(values))
	for name, quantity := range values {
		resource[name] = quantity
	}
	return resource
}

// Add returns the vector sum without modifying either operand.
func (r Resource) Add(other Resource) Resource {
	result := NewResource(r)
	for name, quantity := range other {
		result[name] += quantity
	}
	return result
}

// Sub returns the vector difference without modifying either operand.
func (r Resource) Sub(other Resource) Resource {
	result := NewResource(r)
	for name, quantity := range other {
		result[name] -= quantity
	}
	return result
}

// Fits reports whether every requested resource is no larger than capacity.
func (r Resource) Fits(capacity Resource) bool {
	for name, quantity := range r {
		if quantity > capacity[name] {
			return false
		}
	}
	return true
}

// DominantShare returns the largest share of any requested resource dimension.
// Dimensions absent from, or zero in, total do not contribute a share.
func (r Resource) DominantShare(total Resource) float64 {
	var dominant float64
	for name, quantity := range r {
		if total[name] <= 0 {
			continue
		}
		if share := quantity / total[name]; share > dominant {
			dominant = share
		}
	}
	return dominant
}
