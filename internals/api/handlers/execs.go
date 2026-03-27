package handlers

import (
	"context"
	"fmt"
	"grpcmongoproject/internals/models"
	"grpcmongoproject/internals/repositories/mongodb"
	"grpcmongoproject/pkg/utils"
	pb "grpcmongoproject/proto/gen"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func (s *Server) AddExecs(ctx context.Context, req *pb.Execs) (*pb.Execs, error) {
	err := req.Validate()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	//Check if any invalid inputs are sent in request
	for _, exec := range req.GetExecs() {
		if exec.Id != "" {
			return nil, status.Error(codes.InvalidArgument, "Invalid Request Payload. Non-empty ID fields are not accepted.")
		}
	}

	addedExecs, err := mongodb.AddExecsToDB(ctx, req.GetExecs())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Execs{Execs: addedExecs}, nil
}

func (s *Server) GetExecs(ctx context.Context, req *pb.GetExecsRequest) (*pb.Execs, error) {
	err := req.Validate()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	err = utils.IsAuthorizedUser(ctx, "admin", "exec")
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	filter, err := buildFilter(req.GetExec(), &models.Exec{})
	if err != nil {
		return nil, utils.ErrorHandler(err, err.Error())
	}
	sortOptions := createSortFields(req.GetSortBy())

	execs, err := mongodb.GetExecsFromDB(ctx, sortOptions, filter, req.GetPageNumber(), req.GetPageSize())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Execs{
		Execs: execs,
	}, nil
}

func (s *Server) UpdateExecs(ctx context.Context, req *pb.Execs) (*pb.Execs, error) {
	err := req.Validate()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	updatedExecs, err := mongodb.UpdateExecsInDB(ctx, req.GetExecs())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Execs{Execs: updatedExecs}, nil
}

func (s *Server) DeleteExecs(ctx context.Context, req *pb.ExecIds) (*pb.DeleteExecsConfirmation, error) {
	err := req.Validate()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	execIdsToDelete := []string{}
	for _, execId := range req.GetIds() {
		execIdsToDelete = append(execIdsToDelete, execId.Id)
	}

	err = mongodb.DeleteExecsFromDB(ctx, execIdsToDelete)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.DeleteExecsConfirmation{
		DeletedIds: execIdsToDelete,
		Status:     "success",
	}, nil
}

func (s *Server) ExecLogin(ctx context.Context, req *pb.ExecLoginRequest) (*pb.ExecLoginResponse, error) {
	err := req.Validate()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if req.Username == "" || req.Password == "" {
		return nil, utils.ErrorHandler(fmt.Errorf("username and password fields are required"), "username and password fields are required")
	}
	exec, err := mongodb.GetExecByUsername(ctx, req.Username)
	if err != nil {
		return nil, utils.ErrorHandler(err, err.Error())
	}

	//If user is present, check if entered password is correct
	err = utils.VerifyPassword(req.Password, exec.Password)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Incorrect password.")
	}

	//check if user is inactive.
	if exec.InactiveStatus {
		return nil, utils.ErrorHandler(fmt.Errorf("Executive is inactive."), "Executive is inactive.")
	}

	tokenString, err := utils.SignToken(exec.Id, exec.Username, exec.Role)
	if err != nil {
		return nil, utils.ErrorHandler(err, err.Error())
	}
	return &pb.ExecLoginResponse{Status: true, Token: tokenString}, nil
}

func (s *Server) ExecUpdatePassword(ctx context.Context, req *pb.ExecUpdatePasswordRequest) (*pb.ExecUpdatePasswordResponse, error) {
	err := req.Validate()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	username, role, err := mongodb.UpdateExecPasswordInDB(ctx, req)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	//Generate new JWT Token
	tokenString, err := utils.SignToken(req.Id, username, role)
	if err != nil {
		return nil, utils.ErrorHandler(err, err.Error())
	}

	return &pb.ExecUpdatePasswordResponse{PasswordUpdatedStatus: true, Token: tokenString}, nil
}

func (s *Server) ExecDeactivate(ctx context.Context, execIds *pb.ExecIds) (*pb.Confirmation, error) {
	err := execIds.Validate()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	confirmation, err := mongodb.ExecsDeactivateInDB(ctx, execIds)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.Confirmation{Confirmation: confirmation}, nil
}

func (s *Server) ExecForgotPassword(ctx context.Context, req *pb.ExecForgotPasswordRequest) (*pb.ExecForgotPasswordResponse, error) {
	err := req.Validate()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if req.Email == "" {
		return nil, utils.ErrorHandler(fmt.Errorf("Email should not be empty."), "Email should not be empty.")
	}

	message, err := mongodb.ForgotPasswordDB(ctx, req.Email)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.ExecForgotPasswordResponse{Confirmation: true, Message: message}, nil
}

func (s *Server) ExecResetPassword(ctx context.Context, req *pb.ExecResetPasswordRequest) (*pb.Confirmation, error) {
	err := req.Validate()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if req.Token == "" || req.ConfirmPassword == "" || req.NewPassword == "" {
		return nil, utils.ErrorHandler(fmt.Errorf("token, new_password and confirm_password are mandatory."), "token, new_password and confirm_password are mandatory.")
	}
	if req.ConfirmPassword != req.NewPassword {
		return nil, utils.ErrorHandler(fmt.Errorf("Passwords not matching."), "Passwords not matching.")
	}

	result, err := mongodb.ResetPasswordInDB(ctx, req)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.Confirmation{Confirmation: result}, nil
}

func (s *Server) ExecLogout(ctx context.Context, req *pb.EmptyRequest) (*pb.ExecLogoutResponse, error) {
	err := req.Validate()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, utils.ErrorHandler(fmt.Errorf("No metdata present in request"), "No metdata present in request")
	}

	authHeader, ok := md["authorization"]
	if !ok {
		return nil, utils.ErrorHandler(fmt.Errorf("Unauthorized access"), "Authorization header not present.")
	}
	tokenString := strings.TrimPrefix(authHeader[0], "Bearer ")
	tokenString = strings.TrimSpace(tokenString)

	expiresAt := ctx.Value("exp")
	expiresAtString := fmt.Sprintf("%v", expiresAt)
	expiryTimeStamp, err := strconv.ParseInt(expiresAtString, 10, 64)
	if err != nil {
		return nil, utils.ErrorHandler(err, "Internal server error.")
	}

	expiryTime := time.Unix(expiryTimeStamp, 0)

	utils.JwtStore.AddTokenToMap(tokenString, expiryTime)

	return &pb.ExecLogoutResponse{LoggedOut: true}, nil
}
