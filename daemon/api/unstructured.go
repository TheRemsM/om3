package api

func (t CapabilityList) GetItems() any {
	return t.Items
}

func (t CapabilityItem) Unstructured() map[string]any {
	return map[string]any{
		"kind": t.Kind,
		"meta": t.Meta.Unstructured(),
		"data": t.Data.Unstructured(),
	}
}

func (t Capability) Unstructured() map[string]any {
	return map[string]any{
		"name": t.Name,
	}
}

func (t ScheduleList) GetItems() any {
	return t.Items
}

func (t ScheduleItem) Unstructured() map[string]any {
	return map[string]any{
		"kind": t.Kind,
		"meta": t.Meta.Unstructured(),
		"data": t.Data.Unstructured(),
	}
}

func (t Schedule) Unstructured() map[string]any {
	return map[string]any{
		"action":              t.Action,
		"schedule":            t.Schedule,
		"key":                 t.Key,
		"last_run_at":         t.LastRunAt,
		"last_run_file":       t.LastRunFile,
		"last_success_file":   t.LastSuccessFile,
		"next_run_at":         t.NextRunAt,
		"require_collector":   t.RequireCollector,
		"require_Provisioned": t.RequireProvisioned,
	}
}

func (t NodeList) GetItems() any {
	return t.Items
}

func (t NodeItem) Unstructured() map[string]any {
	return map[string]any{
		"kind": t.Kind,
		"meta": t.Meta.Unstructured(),
		"data": t.Data.Unstructured(),
	}
}

func (t NodeMeta) Unstructured() map[string]any {
	return map[string]any{
		"node": t.Node,
	}
}

func (t Node) Unstructured() map[string]any {
	return map[string]any{
		"config":  t.Config.Unstructured(),
		"monitor": t.Monitor.Unstructured(),
		"status":  t.Status.Unstructured(),
	}
}

func (t InstanceList) GetItems() any {
	return t.Items
}

func (t Instance) Unstructured() map[string]any {
	m := map[string]any{}
	if t.Config != nil {
		m["config"] = t.Config.Unstructured()
	}
	if t.Monitor != nil {
		m["monitor"] = t.Monitor.Unstructured()
	}
	if t.Status != nil {
		m["status"] = t.Status.Unstructured()
	}
	return m
}

func (t InstanceMap) Unstructured() map[string]any {
	m := make(map[string]any)
	for k, v := range t {
		m[k] = v.Unstructured()
	}
	return m
}

func (t InstanceMeta) Unstructured() map[string]any {
	return map[string]any{
		"node":   t.Node,
		"object": t.Object,
	}
}

func (t InstanceItem) Unstructured() map[string]any {
	return map[string]any{
		"kind": t.Kind,
		"meta": t.Meta.Unstructured(),
		"data": t.Data.Unstructured(),
	}
}

func (t KeywordList) GetItems() any {
	return t.Items
}

func (t KeywordItem) Unstructured() map[string]any {
	return map[string]any{
		"kind": t.Kind,
		"meta": t.Meta.Unstructured(),
		"data": t.Data.Unstructured(),
	}
}

func (t KeywordData) Unstructured() any {
	return map[string]any{
		"value": t.Value,
	}
}

func (t KeywordMeta) Unstructured() map[string]any {
	return map[string]any{
		"node":         t.Node,
		"object":       t.Object,
		"keyword":      t.Keyword,
		"is_evaluated": t.IsEvaluated,
		"evaluated_as": t.EvaluatedAs,
	}
}

func (t ObjectList) GetItems() any {
	return t.Items
}

func (t ObjectItem) Unstructured() map[string]any {
	return map[string]any{
		"kind": t.Kind,
		"meta": t.Meta.Unstructured(),
		"data": t.Data.Unstructured(),
	}
}

func (t ObjectMeta) Unstructured() map[string]any {
	return map[string]any{
		"object": t.Object,
	}
}

func (t ObjectData) Unstructured() map[string]any {
	m := map[string]any{
		"avail":              t.Avail,
		"flex_max":           t.FlexMax,
		"flex_min":           t.FlexMin,
		"flex_target":        t.FlexTarget,
		"frozen":             t.Frozen,
		"instances":          t.Instances.Unstructured(),
		"orchestrate":        t.Orchestrate,
		"overall":            t.Overall,
		"placement_policy":   t.PlacementPolicy,
		"placement_state":    t.PlacementState,
		"priority":           t.Priority,
		"provisioned":        t.Provisioned,
		"scope":              t.Scope,
		"topology":           t.Topology,
		"up_instances_count": t.UpInstancesCount,
		"updated_at":         t.UpdatedAt,
	}
	if t.Pool != nil {
		m["pool"] = *t.Pool
	}
	if t.Size != nil {
		m["size"] = *t.Size
	}
	return m
}

func (t ResourceList) GetItems() any {
	return t.Items
}

func (t Resource) Unstructured() map[string]any {
	return map[string]any{
		"config":  t.Config.Unstructured(),
		"monitor": t.Monitor.Unstructured(),
		"status":  t.Status.Unstructured(),
	}
}

func (t ResourceMeta) Unstructured() map[string]any {
	return map[string]any{
		"node":   t.Node,
		"object": t.Object,
		"rid":    t.RID,
	}
}

func (t ResourceItem) Unstructured() map[string]any {
	return map[string]any{
		"kind": t.Kind,
		"meta": t.Meta.Unstructured(),
		"data": t.Data.Unstructured(),
	}
}

func (t NetworkIPList) GetItems() any {
	return t.Items
}

func (t NetworkIP) Unstructured() map[string]any {
	return map[string]any{
		"ip":      t.IP,
		"network": t.Network,
		"node":    t.Node,
		"path":    t.Path,
		"rid":     t.RID,
	}
}

func (t NetworkList) GetItems() any {
	return t.Items
}

func (t Network) Unstructured() map[string]any {
	return map[string]any{
		"errors":  t.Errors,
		"name":    t.Name,
		"network": t.Network,
		"free":    t.Free,
		"size":    t.Size,
		"type":    t.Type,
		"used":    t.Used,
	}
}

func (t PoolList) GetItems() any {
	return t.Items
}

func (t Pool) Unstructured() map[string]any {
	return map[string]any{
		"type":         t.Type,
		"name":         t.Name,
		"capabilities": t.Capabilities,
		"head":         t.Head,
		"errors":       t.Errors,
		"volume_count": t.VolumeCount,
		"free":         t.Free,
		"used":         t.Used,
		"size":         t.Size,
	}
}

func (t PoolVolumeList) GetItems() any {
	return t.Items
}

func (t PoolVolume) Unstructured() map[string]any {
	return map[string]any{
		"pool":      t.Pool,
		"path":      t.Path,
		"children":  t.Children,
		"is_orphan": t.IsOrphan,
		"size":      t.Size,
	}
}
