package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/mickael-carl/sophons/pkg/exec"
	"github.com/mickael-carl/sophons/pkg/proto"
	"github.com/mickael-carl/sophons/pkg/util"
	"github.com/mickael-carl/sophons/pkg/variables"
)

var port = flag.Int("port", 50051, "gRPC server port")

type taskExecuterServer struct {
	proto.UnimplementedTaskExecuterServer
	logger *zap.Logger
}

// TaskExecute implements the gRPC TaskExecuter service.
// It accepts a task with execution context and returns the result.
func (s *taskExecuterServer) TaskExecute(
	ctx context.Context,
	req *proto.TaskExecuteRequest,
) (*proto.TaskExecuteResponse, error) {
	s.logger.Info("received task execution request",
		zap.String("task_name", req.Task.GetName()))

	// 1. Create temp workspace for task execution
	workDir, err := os.MkdirTemp("", "sophons-grpc-exec-*")
	if err != nil {
		return &proto.TaskExecuteResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to create temp workspace: %v", err),
		}, nil
	}
	defer os.RemoveAll(workDir)

	s.logger.Debug("created workspace", zap.String("path", workDir))

	// 2. Extract file archive if provided
	if len(req.FileArchive) > 0 {
		s.logger.Debug("extracting file archive",
			zap.Int("size_bytes", len(req.FileArchive)))

		if err := util.UntarBytes(req.FileArchive, workDir); err != nil {
			return &proto.TaskExecuteResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to extract file archive: %v", err),
			}, nil
		}
	}

	// 3. Setup variable context
	vars, err := convertProtoStructToMap(req.Variables)
	if err != nil {
		return &proto.TaskExecuteResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to convert variables: %v", err),
		}, nil
	}

	s.logger.Debug("setup variables", zap.Int("count", len(vars)))
	execCtx := variables.NewContext(ctx, variables.Variables(vars))

	// 4. Resolve parent path relative to workspace
	// The archive contains files relative to the parent of the source directory.
	// For example, if archiving /tmp/sophons-123/playbooks/, the archive contains
	// entries like "playbooks/files/somefile". So we need to join the workspace
	// with just the base directory name, not the full absolute path.
	parentPath := workDir
	if req.ParentPath != "" {
		parentPath = filepath.Join(workDir, filepath.Base(req.ParentPath))
	}

	s.logger.Debug("resolved parent path",
		zap.String("parent_path", parentPath),
		zap.Bool("is_role", req.IsRole))

	// 5. Convert proto task to exec task
	execTask, err := exec.FromProto(req.Task)
	if err != nil {
		return &proto.TaskExecuteResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to convert task: %v", err),
		}, nil
	}

	// Check if task has valid content
	if execTask.Content == nil {
		return &proto.TaskExecuteResponse{
			Success: false,
			Error:   "task has no content",
		}, nil
	}

	s.logger.Info("executing task",
		zap.String("task_name", execTask.Name),
		zap.String("parent_path", parentPath))

	// 6. Execute task and get result
	result, err := exec.ExecuteTaskWithResult(execCtx, s.logger, *execTask, parentPath, req.IsRole)

	// 7. Build response
	resp := &proto.TaskExecuteResponse{
		Success: err == nil,
	}

	if err != nil {
		s.logger.Error("task execution failed",
			zap.String("task_name", execTask.Name),
			zap.Error(err))
		resp.Error = err.Error()
	} else {
		// Convert result to proto Struct
		resultMap, err := resultToMap(result)
		if err != nil {
			s.logger.Error("failed to convert result to map",
				zap.Error(err))
			return &proto.TaskExecuteResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to convert result: %v", err),
			}, nil
		}

		resp.Result, err = convertMapToProtoStruct(resultMap)
		if err != nil {
			s.logger.Error("failed to convert result to proto struct",
				zap.Error(err))
			return &proto.TaskExecuteResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to convert result to proto: %v", err),
			}, nil
		}

		s.logger.Info("task executed successfully",
			zap.String("task_name", execTask.Name),
			zap.Bool("changed", result.IsChanged()),
			zap.Bool("failed", result.IsFailed()),
			zap.Bool("skipped", result.IsSkipped()))
	}

	return resp, nil
}

func main() {
	flag.Parse()

	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync() //nolint:errcheck

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	proto.RegisterTaskExecuterServer(grpcServer, &taskExecuterServer{logger: logger})

	logger.Info("gRPC task executer listening", zap.Int("port", *port))

	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatal("failed to serve", zap.Error(err))
	}
}
