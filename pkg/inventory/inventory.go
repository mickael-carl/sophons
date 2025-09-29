package inventory

import "maps"

type Inventory struct {
	Groups map[string]Group `yaml:",inline"`
}

type Group struct {
	Hosts    map[string]Variables
	Vars     Variables
	Children map[string]Group
}

type Variables map[string]any

func (i Inventory) Find(node string) map[string]struct{} {
	// All nodes are part of the `all` group inconditionally.
	foundIn := map[string]struct{}{"all": struct{}{}}
	for name, group := range i.Groups {
		maps.Copy(foundIn, group.Find(name, node))
	}
	return foundIn
}

// All returns all the nodes in the automatic "all" group.
func (i Inventory) All() map[string]struct{} {
	all := map[string]struct{}{}
	for _, g := range i.Groups {
		maps.Copy(all, g.All())
	}
	return all
}

func (g Group) Find(groupName, node string) map[string]struct{} {
	groups := map[string]struct{}{}
	if _, ok := g.Hosts[node]; ok {
		groups[groupName] = struct{}{}
	}

	for childName, child := range g.Children {
		foundInChildren := child.Find(childName, node)
		maps.Copy(groups, foundInChildren)
		// In case we found the node in children then it's also in the current
		// group, since that's how Ansible defines group memberships: any host
		// that is a member of a child group is automatically a member of the
		// parent group.
		if len(foundInChildren) > 0 {
			groups[groupName] = struct{}{}
		}
	}

	return groups
}

// All returns all the nodes in this group.
func (g Group) All() map[string]struct{} {
	all := map[string]struct{}{}
	for n, _ := range g.Hosts {
		all[n] = struct{}{}
	}

	for _, child := range g.Children {
		maps.Copy(all, child.All())
	}

	return all
}
