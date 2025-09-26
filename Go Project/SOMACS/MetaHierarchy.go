package SOMACS

import (
	"github.com/google/uuid"
	"strconv"
)

type MetaHierarchy struct {
	RootNodes []*MetaHierarchyNode
}

type MetaHierarchyNode struct {
	Id       uuid.UUID
	Children []*MetaHierarchyNode
	Parent   *MetaHierarchyNode
}

func (mh *MetaHierarchy) createMetaHierarchy(modelAgents []uuid.UUID) {
	mh.RootNodes = make([]*MetaHierarchyNode, len(modelAgents))
	for j, id := range modelAgents {
		mh.RootNodes[j] = &MetaHierarchyNode{
			id,
			make([]*MetaHierarchyNode, 0),
			nil,
		}
	}
}

func (mh *MetaHierarchy) removeFromRootNodes(id uuid.UUID) {
	k := -1
	for j, node := range mh.RootNodes {
		if node.Id == id {
			k = j
			break
		}
	}
	if k == -1 {
		return
	}
	mh.RootNodes = append(mh.RootNodes[:k], mh.RootNodes[k+1:]...)
}

func (mh *MetaHierarchy) subsume(metaAgent uuid.UUID, subsumedAgents []uuid.UUID) {
	children := make([]*MetaHierarchyNode, 0, len(subsumedAgents))
	for _, id := range subsumedAgents {
		child, ok := mh.GetNodeByID(id)
		if ok {
			children = append(children, child)
			mh.removeFromRootNodes(id)
		}
	}
	node := &MetaHierarchyNode{
		metaAgent,
		children,
		nil,
	}
	for _, child := range children {
		child.Parent = node
	}
	mh.RootNodes = append(mh.RootNodes, node)
}

func (mh *MetaHierarchy) dissolve(metaAgent uuid.UUID) {
	node, ok := mh.GetNodeByID(metaAgent)
	if !ok {
		return
	}
	for _, child := range node.Children {
		mh.RootNodes = append(mh.RootNodes, child)
		child.Parent = nil
	}
	if node.Parent == nil {
		mh.removeFromRootNodes(metaAgent)
	} else {
		node.Parent.removeFromChildren(node)
	}
}

func (mh *MetaHierarchy) addAgent(agent uuid.UUID) {
	node := &MetaHierarchyNode{agent, make([]*MetaHierarchyNode, 0), nil}
	mh.RootNodes = append(mh.RootNodes, node)
}

func (mn *MetaHierarchyNode) returnChildNodeWithID(id uuid.UUID) (*MetaHierarchyNode, bool) {
	if id == mn.Id {
		return mn, true
	}
	if mn.Children == nil || len(mn.Children) == 0 {
		return nil, false
	}
	for _, child := range mn.Children {
		ret, ok := child.returnChildNodeWithID(id)
		if ok {
			return ret, ok
		}
	}
	return nil, false
}

func (mn *MetaHierarchyNode) removeFromChildren(node *MetaHierarchyNode) {
	k := -1
	for j, child := range mn.Children {
		if child == node {
			k = j
			break
		}
	}
	if k == -1 {
		return
	}
	mn.Children = append(mn.Children[:k], mn.Children[k+1:]...)
}

func (mn *MetaHierarchyNode) toStringVerbose(tabulators string) string {
	ret := ""
	ret += tabulators + "∟" + mn.Id.String() + "\n"
	if mn.Children == nil || len(mn.Children) == 0 {
		return ret
	}
	for _, child := range mn.Children {
		ret += child.toStringVerbose(tabulators + "\t")
	}
	return ret
}

func (mn *MetaHierarchyNode) toStringCompact(tabulators string) string {
	ret := ""
	if mn.Children == nil || len(mn.Children) == 0 {
		return ret
	}
	ret += tabulators + "∟" + mn.Id.String() + "\n"
	modelAgentCount := 0
	for _, child := range mn.Children {
		if child.Children == nil || len(child.Children) == 0 {
			modelAgentCount++
			continue
		}
		ret += child.toStringCompact(tabulators + "\t")
	}
	if modelAgentCount > 0 {
		ret += tabulators + "\t∟[...] (" + strconv.Itoa(modelAgentCount) + " model agents)\n"
	}
	return ret
}

// Exposed Functions

func (mh *MetaHierarchy) GetNodeByID(id uuid.UUID) (*MetaHierarchyNode, bool) {
	for _, node := range mh.RootNodes {
		ret, ok := node.returnChildNodeWithID(id)
		if ok {
			return ret, ok
		}
	}
	return nil, false
}

func (mh *MetaHierarchy) ToStringVerbose() string {
	ret := ""
	for _, mn := range mh.RootNodes {
		ret += mn.toStringVerbose("")
	}
	return ret
}

func (mh *MetaHierarchy) ToStringCompact() string {
	ret := ""
	modelAgentCount := 0
	for _, mn := range mh.RootNodes {
		if mn.Children == nil || len(mn.Children) == 0 {
			modelAgentCount++
			continue
		}
		ret += mn.toStringCompact("")
	}
	if modelAgentCount > 0 {
		ret += "∟[...] (" + strconv.Itoa(modelAgentCount) + " model agents)\n"
	}
	return ret
}
