package db

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type GraphReader struct {
	client *Neo4jClient
}

func NewGraphReader(client *Neo4jClient) *GraphReader {
	return &GraphReader{client: client}
}

type FileNode struct {
	ID        string        `json:"id"`
	Path      string        `json:"path"`
	Language  string        `json:"language"`
	Functions []FunctionRef `json:"functions"`
}

type FunctionRef struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Signature string `json:"signature"`
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
}

type GraphData struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type GraphNode struct {
	ID    string         `json:"id"`
	Label string         `json:"label"`
	Type  string         `json:"type"`
	Props map[string]any `json:"props"`
}

type GraphEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

// GetFileTree returns all files with their functions for a repository
func (r *GraphReader) GetFileTree(ctx context.Context, repoID string) ([]FileNode, error) {
	result, err := r.client.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (r:Repository {id: $repoId})-[:CONTAINS]->(f:File)
			OPTIONAL MATCH (f)-[:DECLARES]->(fn:Function|Method)
			WITH f, fn
			ORDER BY fn.startLine
			WITH f, collect({
				id: fn.id,
				name: fn.name,
				signature: fn.signature,
				startLine: fn.startLine,
				endLine: fn.endLine
			}) as functions
			RETURN f.id as id, f.path as path, f.language as language, functions
			ORDER BY f.path
		`
		records, err := tx.Run(ctx, query, map[string]any{"repoId": repoID})
		if err != nil {
			return nil, err
		}

		var files []FileNode
		for records.Next(ctx) {
			rec := records.Record()

			// Get basic file info
			id, _ := rec.Get("id")
			path, _ := rec.Get("path")
			language, _ := rec.Get("language")
			functionsRaw, _ := rec.Get("functions")

			file := FileNode{
				ID:        id.(string),
				Path:      path.(string),
				Language:  language.(string),
				Functions: []FunctionRef{},
			}

			// Parse functions
			if functionsRaw != nil {
				functionsList := functionsRaw.([]any)
				for _, fnRaw := range functionsList {
					fnMap := fnRaw.(map[string]any)

					// Skip nil entries (from OPTIONAL MATCH when no functions exist)
					if fnMap["id"] == nil {
						continue
					}

					fn := FunctionRef{
						ID:        fnMap["id"].(string),
						Name:      fnMap["name"].(string),
						Signature: fnMap["signature"].(string),
						StartLine: int(fnMap["startLine"].(int64)),
						EndLine:   int(fnMap["endLine"].(int64)),
					}
					file.Functions = append(file.Functions, fn)
				}
			}

			files = append(files, file)
		}

		if err := records.Err(); err != nil {
			return nil, err
		}

		return files, nil
	})

	if err != nil {
		return nil, err
	}
	return result.([]FileNode), nil
}

// GetGraph returns graph data for visualization
func (r *GraphReader) GetGraph(ctx context.Context, repoID, graphType string) (*GraphData, error) {
	var query string

	if graphType == "calls" {
		// Call graph: show functions and their call relationships
		query = `
			MATCH (r:Repository {id: $repoId})-[:CONTAINS]->(f:File)-[:DECLARES]->(fn:Function|Method)
			OPTIONAL MATCH (fn)-[c:CALLS]->(target:Function|Method)
			RETURN fn, f, c, target
		`
	} else {
		// Structure graph: show files and the functions they declare
		query = `
			MATCH (r:Repository {id: $repoId})-[:CONTAINS]->(f:File)
			OPTIONAL MATCH (f)-[:DECLARES]->(fn:Function|Method)
			RETURN f, fn, null as c, null as target
		`
	}

	result, err := r.client.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, query, map[string]any{"repoId": repoID})
		if err != nil {
			return nil, err
		}

		nodesMap := make(map[string]GraphNode)
		edgesMap := make(map[string]GraphEdge)

		for records.Next(ctx) {
			rec := records.Record()

			if graphType == "calls" {
				// Process call graph
				fnRaw, _ := rec.Get("fn")
				if fnRaw != nil {
					fnNode := fnRaw.(neo4j.Node)
					fnProps := fnNode.GetProperties()

					nodeID := fnProps["id"].(string)
					if _, exists := nodesMap[nodeID]; !exists {
						nodesMap[nodeID] = GraphNode{
							ID:    nodeID,
							Label: fnProps["name"].(string),
							Type:  "Function",
							Props: map[string]any{
								"signature": fnProps["signature"],
								"filePath":  fnProps["filePath"],
							},
						}
					}
				}

				targetRaw, _ := rec.Get("target")
				if targetRaw != nil {
					targetNode := targetRaw.(neo4j.Node)
					targetProps := targetNode.GetProperties()

					targetID := targetProps["id"].(string)
					if _, exists := nodesMap[targetID]; !exists {
						nodesMap[targetID] = GraphNode{
							ID:    targetID,
							Label: targetProps["name"].(string),
							Type:  "Function",
							Props: map[string]any{
								"signature": targetProps["signature"],
								"filePath":  targetProps["filePath"],
							},
						}
					}

					// Add call edge
					callRaw, _ := rec.Get("c")
					if callRaw != nil {
						fnRaw, _ := rec.Get("fn")
						fnNode := fnRaw.(neo4j.Node)
						fnProps := fnNode.GetProperties()

						edgeID := fmt.Sprintf("%s->%s", fnProps["id"].(string), targetID)
						if _, exists := edgesMap[edgeID]; !exists {
							edgesMap[edgeID] = GraphEdge{
								ID:     edgeID,
								Source: fnProps["id"].(string),
								Target: targetID,
								Type:   "CALLS",
							}
						}
					}
				}
			} else {
				// Process structure graph
				fileRaw, _ := rec.Get("f")
				if fileRaw != nil {
					fileNode := fileRaw.(neo4j.Node)
					fileProps := fileNode.GetProperties()

					fileID := fileProps["id"].(string)
					if _, exists := nodesMap[fileID]; !exists {
						nodesMap[fileID] = GraphNode{
							ID:    fileID,
							Label: fileProps["path"].(string),
							Type:  "File",
							Props: map[string]any{
								"language": fileProps["language"],
							},
						}
					}
				}

				fnRaw, _ := rec.Get("fn")
				if fnRaw != nil {
					fnNode := fnRaw.(neo4j.Node)
					fnProps := fnNode.GetProperties()

					fnID := fnProps["id"].(string)
					if _, exists := nodesMap[fnID]; !exists {
						nodesMap[fnID] = GraphNode{
							ID:    fnID,
							Label: fnProps["name"].(string),
							Type:  "Function",
							Props: map[string]any{
								"signature": fnProps["signature"],
							},
						}
					}

					// Add DECLARES edge
					fileRaw, _ := rec.Get("f")
					fileNode := fileRaw.(neo4j.Node)
					fileProps := fileNode.GetProperties()
					fileID := fileProps["id"].(string)

					edgeID := fmt.Sprintf("%s->%s", fileID, fnID)
					if _, exists := edgesMap[edgeID]; !exists {
						edgesMap[edgeID] = GraphEdge{
							ID:     edgeID,
							Source: fileID,
							Target: fnID,
							Type:   "DECLARES",
						}
					}
				}
			}
		}

		if err := records.Err(); err != nil {
			return nil, err
		}

		// Convert maps to slices
		nodes := make([]GraphNode, 0, len(nodesMap))
		for _, node := range nodesMap {
			nodes = append(nodes, node)
		}

		edges := make([]GraphEdge, 0, len(edgesMap))
		for _, edge := range edgesMap {
			edges = append(edges, edge)
		}

		return &GraphData{
			Nodes: nodes,
			Edges: edges,
		}, nil
	})

	if err != nil {
		return nil, err
	}
	return result.(*GraphData), nil
}

// NodeDetail represents detailed information about a node
type NodeDetail struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"` // "File", "Function", or "Method"
	Signature string   `json:"signature,omitempty"`
	FilePath  string   `json:"filePath,omitempty"`
	StartLine int      `json:"startLine,omitempty"`
	EndLine   int      `json:"endLine,omitempty"`
	Calls     []string `json:"calls,omitempty"`     // names of functions this node calls
	CalledBy  []string `json:"calledBy,omitempty"`  // names of functions that call this node
}

// GetNodeDetail returns detailed information about a specific node
func (r *GraphReader) GetNodeDetail(ctx context.Context, repoID, nodeID string) (*NodeDetail, error) {
	result, err := r.client.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// First, get the node details
		query := `
			MATCH (r:Repository {id: $repoId})-[:CONTAINS*1..2]->(node)
			WHERE node.id = $nodeId
			OPTIONAL MATCH (node)-[:CALLS]->(target:Function|Method)
			OPTIONAL MATCH (caller:Function|Method)-[:CALLS]->(node)
			RETURN node,
			       labels(node) as labels,
			       collect(DISTINCT target.name) as calls,
			       collect(DISTINCT caller.name) as calledBy
		`
		records, err := tx.Run(ctx, query, map[string]any{
			"repoId": repoID,
			"nodeId": nodeID,
		})
		if err != nil {
			return nil, err
		}

		if !records.Next(ctx) {
			return nil, nil // Node not found
		}

		rec := records.Record()

		// Get node
		nodeRaw, _ := rec.Get("node")
		if nodeRaw == nil {
			return nil, nil
		}

		node := nodeRaw.(neo4j.Node)
		props := node.GetProperties()

		// Get labels
		labelsRaw, _ := rec.Get("labels")
		labels := labelsRaw.([]any)

		var nodeType string
		for _, label := range labels {
			labelStr := label.(string)
			if labelStr == "File" || labelStr == "Function" || labelStr == "Method" {
				nodeType = labelStr
				break
			}
		}

		detail := &NodeDetail{
			ID:   props["id"].(string),
			Type: nodeType,
		}

		// Set name based on type
		if nameVal, ok := props["name"]; ok && nameVal != nil {
			detail.Name = nameVal.(string)
		} else if pathVal, ok := props["path"]; ok && pathVal != nil {
			detail.Name = pathVal.(string)
		}

		// Set optional fields based on node type
		if nodeType == "Function" || nodeType == "Method" {
			if sig, ok := props["signature"]; ok && sig != nil {
				detail.Signature = sig.(string)
			}
			if fp, ok := props["filePath"]; ok && fp != nil {
				detail.FilePath = fp.(string)
			}
			if sl, ok := props["startLine"]; ok && sl != nil {
				detail.StartLine = int(sl.(int64))
			}
			if el, ok := props["endLine"]; ok && el != nil {
				detail.EndLine = int(el.(int64))
			}

			// Get calls
			callsRaw, _ := rec.Get("calls")
			if callsRaw != nil {
				callsList := callsRaw.([]any)
				for _, call := range callsList {
					if call != nil {
						detail.Calls = append(detail.Calls, call.(string))
					}
				}
			}

			// Get calledBy
			calledByRaw, _ := rec.Get("calledBy")
			if calledByRaw != nil {
				calledByList := calledByRaw.([]any)
				for _, caller := range calledByList {
					if caller != nil {
						detail.CalledBy = append(detail.CalledBy, caller.(string))
					}
				}
			}
		} else if nodeType == "File" {
			if path, ok := props["path"]; ok && path != nil {
				detail.FilePath = path.(string)
			}
		}

		if err := records.Err(); err != nil {
			return nil, err
		}

		return detail, nil
	})

	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.(*NodeDetail), nil
}
