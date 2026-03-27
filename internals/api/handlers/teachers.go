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

func (s *Server) AddTeachers(ctx context.Context, req *pb.Teachers) (*pb.Teachers, error) {
	//Check if any invalid inputs are sent in request
	for _, teacher := range req.GetTeachers() {
		if teacher.Id != "" {
			return nil, status.Error(codes.InvalidArgument, "Invalid Request Payload. Non-empty ID fields are not accepted.")
		}
	}

	addedTeachers, err := mongodb.AddTeachersToDB(ctx, req.GetTeachers())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Teachers{Teachers: addedTeachers}, nil
}

func (s *Server) GetTeachers(ctx context.Context, req *pb.GetTeachersRequest) (*pb.Teachers, error) {
	filter, err := buildFilter(req.GetTeacher(), &models.Teacher{})
	if err != nil {
		return nil, utils.ErrorHandler(err, err.Error())
	}
	sortOptions := createSortFields(req.GetSortBy())

	teachers, err := mongodb.GetTeachersFromDB(ctx, sortOptions, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Teachers{
		Teachers: teachers,
	}, nil
}

func (s *Server) UpdateTeachers(ctx context.Context, req *pb.Teachers) (*pb.Teachers, error) {
	updatedTeachers, err := mongodb.UpdateTeachersInDB(ctx, req.GetTeachers())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Teachers{Teachers: updatedTeachers}, nil
}

func (s *Server) DeleteTeachers(ctx context.Context, req *pb.TeacherIds) (*pb.DeleteTeachersConfirmation, error) {
	teacherIdsToDelete := []string{}
	for _, teacherId := range req.GetIds() {
		teacherIdsToDelete = append(teacherIdsToDelete, teacherId.Id)
	}

	err := mongodb.DeleteTeachersFromDB(ctx, teacherIdsToDelete)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.DeleteTeachersConfirmation{
		DeletedIds: teacherIdsToDelete,
		Status:     "success",
	}, nil
}

func (s *Server) GetStudentsByClassTeacher(ctx context.Context, teacherId *pb.TeacherId) (*pb.Students, error) {
	pbStudents, err := mongodb.GetStudentsByClassTeacherFromDB(ctx, teacherId.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.Students{Students: pbStudents}, nil
}

func (s *Server) GetStudentsCountByClassTeacher(ctx context.Context, teacherId *pb.TeacherId) (*pb.StudentsCount, error) {
	noOfStudents, err := mongodb.GetStudentsCountByClassTeacherFromDB(ctx, teacherId.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.StudentsCount{Status: true, StudentsCount: int32(noOfStudents)}, nil
}


