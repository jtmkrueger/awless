/*
Copyright 2017 WALLIX

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ast

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
)

type Node interface {
	clone() Node
	String() string
}

type Statement struct {
	Node
	Result interface{}
	Line   string
	Err    error
}

func (s *Statement) clone() *Statement {
	newStat := &Statement{}
	newStat.Node = s.Node.clone()
	newStat.Result = s.Result
	newStat.Err = s.Err

	return newStat
}

func (s *Statement) Action() string {
	switch n := s.Node.(type) {
	case *ExpressionNode:
		return n.Action
	case *DeclarationNode:
		return n.Right.Action
	default:
		panic(fmt.Sprintf("unknown type of node %T", s.Node))
	}
}

func (s *Statement) Entity() string {
	switch n := s.Node.(type) {
	case *ExpressionNode:
		return n.Entity
	case *DeclarationNode:
		return n.Right.Entity
	default:
		panic(fmt.Sprintf("unknown type of node %T", s.Node))
	}
}

func (s *Statement) Params() map[string]interface{} {
	switch n := s.Node.(type) {
	case *ExpressionNode:
		return n.Params
	case *DeclarationNode:
		return n.Right.Params
	default:
		panic(fmt.Sprintf("unknown type of node %T", s.Node))
	}
}

type AST struct {
	Statements []*Statement

	currentStatement *Statement
	currentKey       string
}

func (a *AST) String() string {
	var all []string
	for _, stat := range a.Statements {
		all = append(all, stat.String())
	}
	return strings.Join(all, "\n")
}

type IdentifierNode struct {
	Ident string
	Val   interface{}
}

func (n *IdentifierNode) clone() Node {
	return &IdentifierNode{
		Ident: n.Ident,
		Val:   n.Val,
	}
}

func (n *IdentifierNode) String() string {
	return fmt.Sprintf("%s", n.Ident)
}

type VarNode struct {
	I    *IdentifierNode
	Hole map[string]string
}

func (n *VarNode) ProcessHoles(fills map[string]interface{}) map[string]interface{} {
	processed := make(map[string]interface{})
	for key, hole := range n.Hole {
		if val, ok := fills[hole]; ok {
			n.I.Val = val
			processed[hole] = val
			delete(n.Hole, key)
		}
	}
	return processed
}

func (n *VarNode) String() string {
	return fmt.Sprintf("var %s = %v", n.I.Ident, n.I.Val)
}

func (n *VarNode) clone() Node {
	return &VarNode{
		I:    n.I.clone().(*IdentifierNode),
		Hole: make(map[string]string),
	}
}

type DeclarationNode struct {
	Left  *IdentifierNode
	Right *ExpressionNode
}

func (n *DeclarationNode) clone() Node {
	return &DeclarationNode{
		Left:  n.Left.clone().(*IdentifierNode),
		Right: n.Right.clone().(*ExpressionNode),
	}
}

func (n *DeclarationNode) String() string {
	return fmt.Sprintf("%s = %s", n.Left, n.Right)
}

type ExpressionNode struct {
	Action, Entity string
	Refs           map[string]string
	Params         map[string]interface{}
	Aliases        map[string]string
	Holes          map[string]string
}

func (n *ExpressionNode) clone() Node {
	expr := &ExpressionNode{
		Action: n.Action, Entity: n.Entity,
		Refs:    make(map[string]string),
		Params:  make(map[string]interface{}),
		Aliases: make(map[string]string),
		Holes:   make(map[string]string),
	}

	for k, v := range n.Refs {
		expr.Refs[k] = v
	}
	for k, v := range n.Params {
		expr.Params[k] = v
	}
	for k, v := range n.Aliases {
		expr.Aliases[k] = v
	}
	for k, v := range n.Holes {
		expr.Holes[k] = v
	}

	return expr
}

func (n *ExpressionNode) String() string {
	var all []string

	refs := sortAndMapString(n.Refs, func(k, v string) string {
		return fmt.Sprintf("%s=$%v", k, v)
	})
	all = append(all, refs...)

	params := sortAndMap(n.Params, func(k string, v interface{}) string {
		return fmt.Sprintf("%s=%v", k, v)
	})
	all = append(all, params...)

	aliases := sortAndMapString(n.Aliases, func(k, v string) string {
		return fmt.Sprintf("%s=@%s", k, v)
	})
	all = append(all, aliases...)

	holes := sortAndMapString(n.Holes, func(k, v string) string {
		return fmt.Sprintf("%s={%s}", k, v)
	})
	all = append(all, holes...)

	sort.Strings(all)

	return fmt.Sprintf("%s %s %s", n.Action, n.Entity, strings.Join(all, " "))
}

// Sort map and apply fn to output printed params always in the same order (useful for tests)
func sortAndMapString(m map[string]string, fn func(k, v string) string) (out []string) {
	newM := make(map[string]interface{})
	for k, v := range m {
		newM[k] = v
	}

	newFn := func(k string, v interface{}) string {
		if s, ok := v.(string); ok {
			return fn(k, s)
		}
		return ""
	}

	return sortAndMap(newM, newFn)
}

func sortAndMap(m map[string]interface{}, fn func(k string, v interface{}) string) (out []string) {
	var keys []string
	for k, _ := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		out = append(out, fn(k, m[k]))
	}
	return
}

func (n *ExpressionNode) ProcessHoles(fills map[string]interface{}) map[string]interface{} {
	processed := make(map[string]interface{})
	for key, hole := range n.Holes {
		if val, ok := fills[hole]; ok {
			if n.Params == nil {
				n.Params = make(map[string]interface{})
			}
			n.Params[key] = val
			processed[key] = val
			delete(n.Holes, key)
		}
	}
	return processed
}

func (n *ExpressionNode) ProcessRefs(fills map[string]interface{}) {
	for key, ref := range n.Refs {
		if val, ok := fills[ref]; ok {
			if n.Params == nil {
				n.Params = make(map[string]interface{})
			}
			n.Params[key] = val
			delete(n.Refs, key)
		}
	}
}

func (s *AST) AddAction(text string) {
	expr := s.currentExpression()
	if expr == nil {
		s.addStatement(&ExpressionNode{Action: text})
	} else {
		expr.Action = text
	}
}

func (s *AST) AddEntity(text string) {
	expr := s.currentExpression()
	expr.Entity = text
}

func (s *AST) AddDeclarationIdentifier(text string) {
	decl := &DeclarationNode{
		Left:  &IdentifierNode{Ident: text},
		Right: &ExpressionNode{},
	}
	s.addStatement(decl)
}

func (s *AST) LineDone() {
	s.currentStatement = nil
	s.currentKey = ""
}

func (s *AST) AddVarIdentifier(text string) {
	vnode := &VarNode{
		I:    &IdentifierNode{Ident: text},
		Hole: make(map[string]string),
	}
	s.addStatement(vnode)
}

func (s *AST) AddVarValue(text string) {
	vnode := s.currentVarDecl()
	vnode.I.Val = text
}

func (s *AST) AddVarIntValue(text string) {
	vnode := s.currentVarDecl()
	vnode.I.Val = parseInt(text)
}

func (s *AST) AddVarCidrValue(text string) {
	vnode := s.currentVarDecl()
	vnode.I.Val = parseCIDR(text)
}

func (s *AST) AddVarIpValue(text string) {
	vnode := s.currentVarDecl()
	vnode.I.Val = parseIP(text)
}

func (s *AST) AddVarHoleValue(text string) {
	vnode := s.currentVarDecl()
	vnode.Hole[vnode.I.Ident] = text
}

func (s *AST) AddParamKey(text string) {
	expr := s.currentExpression()
	if expr.Params == nil {
		expr.Refs = make(map[string]string)
		expr.Params = make(map[string]interface{})
		expr.Aliases = make(map[string]string)
		expr.Holes = make(map[string]string)
	}
	s.currentKey = text
}

func (s *AST) AddParamValue(text string) {
	expr := s.currentExpression()
	expr.Params[s.currentKey] = text
}

func (s *AST) AddParamIntValue(text string) {
	expr := s.currentExpression()
	expr.Params[s.currentKey] = parseInt(text)
}

func (s *AST) AddParamCidrValue(text string) {
	expr := s.currentExpression()
	expr.Params[s.currentKey] = parseCIDR(text)
}

func (s *AST) AddParamIpValue(text string) {
	expr := s.currentExpression()
	expr.Params[s.currentKey] = parseIP(text)
}

func (s *AST) AddParamRefValue(text string) {
	expr := s.currentExpression()
	expr.Refs[s.currentKey] = text
}

func (s *AST) AddParamAliasValue(text string) {
	expr := s.currentExpression()
	expr.Aliases[s.currentKey] = text
}

func (s *AST) AddParamHoleValue(text string) {
	expr := s.currentExpression()
	expr.Holes[s.currentKey] = text
}

func (s *AST) currentExpression() *ExpressionNode {
	st := s.currentStatement
	if st == nil {
		return nil
	}

	switch st.Node.(type) {
	case *ExpressionNode:
		return st.Node.(*ExpressionNode)
	case *DeclarationNode:
		return st.Node.(*DeclarationNode).Right
	default:
		panic("last expression: unexpected node type")
	}
}

func (s *AST) currentVarDecl() *VarNode {
	st := s.currentStatement
	if st == nil {
		return nil
	}

	switch st.Node.(type) {
	case *VarNode:
		return st.Node.(*VarNode)
	default:
		panic("expected var node type")
	}
}

func (a *AST) ExecutionStatements() (out []*Statement) {
	for _, sts := range a.Statements {
		if _, ok := sts.Node.(*VarNode); ok {
			continue
		}
		out = append(out, sts)
	}
	return
}

func (a *AST) Clone() *AST {
	clone := &AST{}
	for _, stat := range a.Statements {
		clone.Statements = append(clone.Statements, stat.clone())
	}
	return clone
}

func (s *AST) addStatement(n Node) {
	stat := &Statement{Node: n}
	s.currentStatement = stat
	s.Statements = append(s.Statements, stat)
}

func parseInt(text string) (num int) {
	num, err := strconv.Atoi(text)
	if err != nil {
		panic(fmt.Sprintf("cannot convert '%s' to int", text))
	}
	return
}

func parseIP(text string) string {
	ip := net.ParseIP(text)
	if ip == nil {
		panic(fmt.Sprintf("cannot convert '%s' to net ip", text))
	}
	return ip.String()
}

func parseCIDR(text string) string {
	_, cidr, err := net.ParseCIDR(text)
	if err != nil {
		panic(fmt.Sprintf("cannot convert '%s' to net cidr", text))
	}
	return cidr.String()
}
