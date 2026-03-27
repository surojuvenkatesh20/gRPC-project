package handlers

import (
	"context"
	"grpcmongoproject/internals/models"
	"grpcmongoproject/internals/repositories/mongodb"
	"grpcmongoproject/pkg/utils"
	pb "grpcmongoproject/proto/gen"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) AddStudents(ctx context.Context, req *pb.Students) (*pb.Students, error) {
	//Check if any invalid inputs are sent in request
	for _, student := range req.GetStudents() {
		if student.Id != "" {
			return nil, status.Error(codes.InvalidArgument, "Invalid Request Payload. Non-empty ID fields are not accepted.")
		}
	}

	addedStudents, err := mongodb.AddStudentsToDB(ctx, req.GetStudents())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Students{Students: addedStudents}, nil
}

func (s *Server) GetStudents(ctx context.Context, req *pb.GetStudentsRequest) (*pb.Students, error) {
	filter, err := buildFilter(req.GetStudent(), &models.Student{})
	if err != nil {
		return nil, utils.ErrorHandler(err, err.Error())
	}
	sortOptions := createSortFields(req.GetSortBy())

	students, err := mongodb.GetStudentsFromDB(ctx, sortOptions, filter, req.GetPageNumber(), req.GetPageSize())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Students{
		Students: students,
	}, nil
}

func (s *Server) UpdateStudents(ctx context.Context, req *pb.Students) (*pb.Students, error) {
	updatedStudents, err := mongodb.UpdateStudentsInDB(ctx, req.GetStudents())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Students{Students: updatedStudents}, nil
}

func (s *Server) DeleteStudents(ctx context.Context, req *pb.StudentIds) (*pb.DeleteStudentsConfirmation, error) {
	studentIdsToDelete := []string{}
	for _, studentId := range req.GetIds() {
		studentIdsToDelete = append(studentIdsToDelete, studentId.Id)
	}

	err := mongodb.DeleteStudentsFromDB(ctx, studentIdsToDelete)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.DeleteStudentsConfirmation{
		DeletedIds: studentIdsToDelete,
		Status:     "success",
	}, nil
}
