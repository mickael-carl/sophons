package inventory

import (
	"maps"

	"github.com/mickael-carl/sophons/pkg/variables"
)

type Inventory struct {
	Groups map[string]Group `yaml:",inline"`
}

type Group struct {
	Hosts    map[string]variables.Variables
	Vars     variables.Variables
	Children map[string]Group
}

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

// TODO: this may need special handling for `all`
func (i Inventory) NodeVars(node string) variables.Variables {
	hostVars := variables.Variables{}
	groupVars := variables.Variables{}
	for _, group := range i.Groups {
		h, g := group.NodeVars(node)
		maps.Copy(hostVars, h)
		maps.Copy(groupVars, g)
	}
	maps.Copy(groupVars, hostVars)

	return groupVars
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
	for n := range g.Hosts {
		all[n] = struct{}{}
	}

	for _, child := range g.Children {
		maps.Copy(all, child.All())
	}

	return all
}

// NodeVars returns host an group variables associated with a node. It does
// return both sets independently because of Ansible's variables merge order:
// host variables have the highest precedence, which means they need to bubble
// all the way up in the inventory.
func (g Group) NodeVars(node string) (variables.Variables, variables.Variables) {
	groupVars := variables.Variables{}
	hostVars := variables.Variables{}
	childrenVars := variables.Variables{}

	if v, ok := g.Hosts[node]; ok {
		maps.Copy(hostVars, v)
		maps.Copy(groupVars, g.Vars)
	}

	for _, child := range g.Children {
		hostInChildVars, childVars := child.NodeVars(node)
		maps.Copy(hostVars, hostInChildVars)
		maps.Copy(childrenVars, childVars)
	}

	if len(childrenVars) > 0 {
		maps.Copy(groupVars, g.Vars)
		maps.Copy(groupVars, childrenVars)
	}

	return hostVars, groupVars
}
