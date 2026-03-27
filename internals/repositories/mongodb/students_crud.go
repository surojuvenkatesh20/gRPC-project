package mongodb

import (
	"context"
	"fmt"
	"grpcmongoproject/internals/models"
	"grpcmongoproject/pkg/utils"

	pb "grpcmongoproject/proto/gen"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func AddStudentsToDB(ctx context.Context, studentsFromReq []*pb.Student) ([]*pb.Student, error) {
	mongoClient, err := CreateMongoClient()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Unable to connect to mongodb.")
	}
	defer mongoClient.Disconnect(ctx)

	//converting each protobuf field into models.Student(bson) field
	newStudents := make([]*models.Student, len(studentsFromReq))
	for i, pbStudent := range studentsFromReq {
		newStudents[i] = MapProtoBufEntityToModelEntity(pbStudent, func() *models.Student { return &models.Student{} })
	}

	//converting each models.Student(bson) field into protobuf fields.
	addedStudents := []*pb.Student{}
	for _, student := range newStudents {
		result, err := mongoClient.Database("school").Collection("students").InsertOne(ctx, student)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Unable to create students.")
		}

		objectId, ok := result.InsertedID.(primitive.ObjectID)
		if ok {
			student.Id = objectId.Hex()
		}
		addedStudents = append(addedStudents, MapModelEntityToProtoBufEntity(student, func() *pb.Student { return &pb.Student{} }))

	}
	return addedStudents, nil
}

func GetStudentsFromDB(ctx context.Context, sortOptions bson.D, filter bson.M, pageNumber, pageSize uint32) ([]*pb.Student, error) {
	mongoClient, err := CreateMongoClient()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Unable to connect to mongodb.")
	}
	defer mongoClient.Disconnect(ctx)

	collection := mongoClient.Database("school").Collection("students")

	var cursor *mongo.Cursor

	if pageNumber < 1 {
		pageNumber = 1
	}
	if pageSize < 10 {
		pageSize = 10
	}
	findOptions := options.Find()
	findOptions.SetSkip(int64((pageNumber - 1) * pageSize))
	findOptions.SetLimit(int64(pageSize))

	if len(sortOptions) > 0 {
		findOptions.SetSort(sortOptions)
	}
	cursor, err = collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, utils.ErrorHandler(err, "Internal Server Error.")
	}

	students, err := DecodeEntities(ctx, cursor, func() *models.Student { return &models.Student{} }, func() *pb.Student { return &pb.Student{} })
	if err != nil {
		return nil, err
	}
	return students, nil
}

func UpdateStudentsInDB(ctx context.Context, students []*pb.Student) ([]*pb.Student, error) {
	mongoClient, err := CreateMongoClient()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Unable to connect to mongo.")
	}
	defer mongoClient.Disconnect(ctx)

	var updatedStudents []*pb.Student
	for _, student := range students {
		if student.Id == "" {
			return nil, utils.ErrorHandler(fmt.Errorf("id should not be empty"), "id should not be empty")
		}
		modelStudent := MapProtoBufEntityToModelEntity(student, func() *models.Student { return &models.Student{} })

		objectId, err := primitive.ObjectIDFromHex(modelStudent.Id)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Internal server error.")
		}

		bytesDoc, err := bson.Marshal(modelStudent)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Internal Server Error")
		}

		var updatedDoc bson.M
		err = bson.Unmarshal(bytesDoc, &updatedDoc)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Internal server error.")
		}
		delete(updatedDoc, "_id")
		_, err = mongoClient.Database("school").Collection("students").UpdateOne(ctx, bson.M{"_id": objectId}, bson.M{"$set": updatedDoc})
		if err != nil {
			return nil, utils.ErrorHandler(err, fmt.Sprintf("Error updating student with id: %s", modelStudent.Id))
		}
		updatedStudents = append(updatedStudents, student)
	}
	return updatedStudents, nil
}

func DeleteStudentsFromDB(ctx context.Context, studentIdsToDelete []string) error {
	mongoClient, err := CreateMongoClient()
	if err != nil {
		return utils.ErrorHandler(err, "unable to connect to mongodb.")
	}
	defer mongoClient.Disconnect(ctx)

	objectIds := make([]primitive.ObjectID, len(studentIdsToDelete))
	for i, id := range studentIdsToDelete {
		if id == "" {
			return utils.ErrorHandler(fmt.Errorf("id field should not be empty."), "id field should not be empty.")
		}
		objectIds[i], err = primitive.ObjectIDFromHex(id)
		if err != nil {
			return utils.ErrorHandler(fmt.Errorf("Incorrect id in request: %s", id), fmt.Sprintf("Incorrect id in request: %s", id))
		}
	}

	filter := bson.M{
		"_id": bson.M{
			"$in": objectIds,
		},
	}

	result, err := mongoClient.Database("school").Collection("students").DeleteMany(ctx, filter)
	if err != nil {
		return utils.ErrorHandler(err, "Error in deleting students.")
	}

	if result.DeletedCount == 0 {
		return utils.ErrorHandler(fmt.Errorf("No Students are deleted."), "No Students are deleted.")
	}
	return nil
}
