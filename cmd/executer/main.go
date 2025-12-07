package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/nikolalohinski/gonja/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"

	"go.uber.org/zap"

	"github.com/mickael-carl/sophons/pkg/exec"
	"github.com/mickael-carl/sophons/pkg/inventory"
	"github.com/mickael-carl/sophons/pkg/playbook"
	"github.com/mickael-carl/sophons/pkg/proto"
	"github.com/mickael-carl/sophons/pkg/role"
	"github.com/mickael-carl/sophons/pkg/util"
	"github.com/mickael-carl/sophons/pkg/variables"
)

var (
	inventoryPath    = flag.String("i", "", "path to inventory file")
	dataArchive      = flag.String("d", "", "path to data archive")
	playbooksDirName = flag.String("p", "", "name of the directory containing playbooks")
	node             = flag.String("n", "localhost", "name of the node to run the playbook against")
	grpcExecuter     = flag.String("grpc-executer", "", "gRPC executer address (e.g., localhost:50051) for distributed execution")
)

// needsFileArchive determines if a task requires file access and should include a file archive.
func needsFileArchive(task *proto.Task) bool {
	switch task.Content.(type) {
	case *proto.Task_Copy:
		return true
	case *proto.Task_Template:
		return true
	case *proto.Task_IncludeTasks:
		return true
	case *proto.Task_ImportTasks:
		return true
	default:
		return false
	}
}

// createFileArchive creates a tar.gz archive of the necessary files for task execution.
func createFileArchive(parentPath string, isRole bool) (string, error) {
	// Determine what to include:
	// - If isRole: include the role directory structure
	// - Otherwise: include the playbook directory
	archivePath, err := util.Tar(parentPath)
	if err != nil {
		return "", fmt.Errorf("failed to create archive: %w", err)
	}

	return archivePath, nil
}

// executeTaskViaGRPC executes a task by calling the gRPC executer service.
func executeTaskViaGRPC(
	ctx context.Context,
	logger *zap.Logger,
	client proto.TaskExecuterClient,
	task *proto.Task,
	parentPath string,
	isRole bool,
) error {
	// 1. Get variables from context
	vars, ok := variables.FromContext(ctx)
	if !ok {
		vars = variables.Variables{}
	}

	// 2. Convert variables to proto Struct
	varsProto, err := structpb.NewStruct(vars)
	if err != nil {
		return fmt.Errorf("failed to convert variables to proto: %w", err)
	}

	// 3. Create file archive if needed
	var archiveBytes []byte
	if needsFileArchive(task) {
		archivePath, err := createFileArchive(parentPath, isRole)
		if err != nil {
			return fmt.Errorf("failed to create file archive: %w", err)
		}
		defer os.Remove(archivePath)

		archiveBytes, err = os.ReadFile(archivePath)
		if err != nil {
			return fmt.Errorf("failed to read archive: %w", err)
		}
	}

	// 4. Make gRPC call with timeout
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	req := &proto.TaskExecuteRequest{
		Task:        task,
		Variables:   varsProto,
		ParentPath:  parentPath,
		IsRole:      isRole,
		FileArchive: archiveBytes,
	}

	logger.Debug("sending gRPC request",
		zap.String("task_name", task.Name),
		zap.Int("archive_size", len(archiveBytes)))

	resp, err := client.TaskExecute(callCtx, req)
	if err != nil {
		return fmt.Errorf("gRPC call failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("task execution failed: %s", resp.Error)
	}

	// 5. Update variables with result if task had register
	if task.Register != "" {
		vars[task.Register] = resp.Result.AsMap()
		logger.Debug("registered variable",
			zap.String("name", task.Register))
	}

	logger.Info("task completed via gRPC",
		zap.String("task_name", task.Name),
		zap.Bool("success", resp.Success))

	return nil
}

func playbookApply(ctx context.Context, logger *zap.Logger, playbookPath, node string, groups map[string]struct{}, roles map[string]role.Role, rolesDir string, grpcClient proto.TaskExecuterClient) error {
	playbookData, err := os.ReadFile(playbookPath)
	if err != nil {
		return fmt.Errorf("failed to read playbook from %s: %w", playbookPath, err)
	}

	var playbook playbook.Playbook
	if err := yaml.Unmarshal(playbookData, &playbook); err != nil {
		return fmt.Errorf("failed to unmarshal playbook from %s: %w", playbookPath, err)
	}

	for _, play := range playbook {
		if _, ok := groups[play.Hosts]; ok || play.Hosts == node {
			inventoryVars, ok := variables.FromContext(ctx)
			if !ok {
				inventoryVars = variables.Variables{}
			}

			playVars := variables.Variables{}
			playVars.Merge(inventoryVars)

			playVars.Merge(play.Vars)

			for _, varsFile := range play.VarsFiles {
				absVarsFilePath := filepath.Join(filepath.Dir(playbookPath), varsFile)
				fileVars, err := variables.LoadFromFile(absVarsFilePath)
				if err != nil {
					return fmt.Errorf("failed to load vars file %s for play: %w", absVarsFilePath, err)
				}
				playVars.Merge(fileVars)
			}

			playCtx := variables.NewContext(ctx, playVars)

			// Ansible executes roles first, then tasks. See
			// https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_reuse_roles.html#using-roles-at-the-play-level.
			for _, roleName := range play.Roles {
				logger.Debug("executing role", zap.String("role", roleName))

				role, ok := roles[roleName]
				if !ok {
					return fmt.Errorf("no such role: %s", roleName)
				}

				// Headsup: roles variables are *not* scoped to only the role
				// itself. This means this call actually *has to mutate*
				// playCtx, so that variables defined in a role can be used in
				// subsequent ones as well as the rest of the play. Sorry
				// Ansible but this is STUPID.
				if err := role.Apply(playCtx, logger, filepath.Join(rolesDir, roleName)); err != nil {
					return fmt.Errorf("failed to apply role %s: %w", roleName, err)
				}
			}
			for _, protoTask := range play.Tasks {
				if grpcClient != nil {
					// gRPC execution path
					if err := executeTaskViaGRPC(
						playCtx,
						logger,
						grpcClient,
						protoTask,
						filepath.Dir(playbookPath),
						false,
					); err != nil {
						return fmt.Errorf("failed to execute task via gRPC: %w", err)
					}
				} else {
					// In-process execution path (current)
					execTask, err := exec.FromProto(protoTask)
					if err != nil {
						return fmt.Errorf("failed to convert task: %w", err)
					}

					if err := exec.ExecuteTask(playCtx, logger, *execTask, filepath.Dir(playbookPath), false); err != nil { // use playCtx
						return fmt.Errorf("failed to execute task: %w", err)
					}
				}
			}
		}
	}
	return nil
}

func main() {
	gonja.DefaultConfig.StrictUndefined = true
	flag.Parse()

	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}
	defer logger.Sync() //nolint:errcheck

	if len(flag.Args()) != 1 {
		logger.Fatal("usage: executer spec.yaml")
	}

	if *dataArchive != "" && *playbooksDirName == "" || *dataArchive == "" && *playbooksDirName != "" {
		logger.Fatal("when either -d or -p is set, both flags must be set")
	}

	groups := map[string]struct{}{"all": {}}
	vars := variables.Variables{}

	if *inventoryPath != "" {
		inventoryData, err := os.ReadFile(*inventoryPath)
		if err != nil {
			logger.Fatal("failed to read inventory", zap.String("path", *inventoryPath), zap.Error(err))
		}

		var inventory inventory.Inventory
		if err := yaml.Unmarshal(inventoryData, &inventory); err != nil {
			logger.Fatal("failed to unmarshal inventory", zap.String("path", *inventoryPath), zap.Error(err))
		}

		groups = inventory.Find(*node)
		vars = inventory.NodeVars(*node)
	}

	ctx := variables.NewContext(context.Background(), vars)

	// Setup gRPC client if flag is provided
	var grpcClient proto.TaskExecuterClient
	var grpcConn *grpc.ClientConn
	if *grpcExecuter != "" {
		conn, err := grpc.NewClient(
			*grpcExecuter,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			logger.Fatal("failed to create gRPC client",
				zap.String("addr", *grpcExecuter),
				zap.Error(err))
		}
		grpcConn = conn
		defer grpcConn.Close()

		grpcClient = proto.NewTaskExecuterClient(conn)
		logger.Info("created gRPC client",
			zap.String("addr", *grpcExecuter))
	}

	playbookDir := filepath.Dir(flag.Args()[0])
	if *dataArchive != "" {
		if err := util.Untar(*dataArchive, filepath.Dir(*dataArchive)); err != nil {
			logger.Fatal("failed to untar archive", zap.String("path", *dataArchive), zap.Error(err))
		}
		playbookDir = filepath.Join(filepath.Dir(*dataArchive), *playbooksDirName)
	}

	rolesDir := filepath.Join(playbookDir, "roles")
	fsys := os.DirFS(rolesDir)

	roles, err := role.DiscoverRoles(fsys)
	if err != nil {
		logger.Fatal("failed to discover roles", zap.Error(err))
	}

	playbookPath := flag.Args()[0]
	if err := playbookApply(ctx, logger, playbookPath, *node, groups, roles, rolesDir, grpcClient); err != nil {
		logger.Fatal("failed to run playbook", zap.String("path", playbookPath), zap.Error(err))
	}
}
