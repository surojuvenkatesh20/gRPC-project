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

func AddTeachersToDB(ctx context.Context, teachersFromReq []*pb.Teacher) ([]*pb.Teacher, error) {
	mongoClient, err := CreateMongoClient()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Unable to connect to mongodb.")
	}
	defer mongoClient.Disconnect(ctx)

	//converting each protobuf field into models.Teacher(bson) field
	newTeachers := make([]*models.Teacher, len(teachersFromReq))
	for i, pbTeacher := range teachersFromReq {
		// newTeachers[i] = MapProtoBufTeacherToModelTeacher(pbTeacher)
		newTeachers[i] = MapProtoBufEntityToModelEntity(pbTeacher, func() *models.Teacher { return &models.Teacher{} })
	}

	//converting each models.Teacher(bson) field into protobuf fields.
	addedTeachers := []*pb.Teacher{}
	for _, teacher := range newTeachers {
		result, err := mongoClient.Database("school").Collection("teachers").InsertOne(ctx, teacher)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Unable to create Teachers.")
		}

		objectId, ok := result.InsertedID.(primitive.ObjectID)
		if ok {
			teacher.Id = objectId.Hex()
		}
		// addedTeachers = append(addedTeachers, MapModelTeacherToProtoBufTeacher(teacher))
		addedTeachers = append(addedTeachers, MapModelEntityToProtoBufEntity(teacher, func() *pb.Teacher { return &pb.Teacher{} }))

	}
	return addedTeachers, nil
}

// func MapModelTeacherToProtoBufTeacher(teacher *models.Teacher) *pb.Teacher {
// 	protoTeacher := &pb.Teacher{}
// 	modelVal := reflect.ValueOf(teacher).Elem()
// 	pbTeacher := reflect.ValueOf(protoTeacher).Elem()

// 	for i := 0; i < modelVal.NumField(); i++ {
// 		field := modelVal.Field(i)
// 		fieldName := modelVal.Type().Field(i).Name

// 		pbField := pbTeacher.FieldByName(fieldName)

// 		if pbField.IsValid() && pbField.CanSet() {
// 			pbField.Set(field)
// 		}
// 	}
// 	return protoTeacher
// }

// func MapProtoBufTeacherToModelTeacher(pbTeacher *pb.Teacher) *models.Teacher {
// 	modelTeacher := models.Teacher{}
// 	pbVal := reflect.ValueOf(pbTeacher).Elem()
// 	modelVal := reflect.ValueOf(&modelTeacher).Elem()

// 	for i := 0; i < pbVal.NumField(); i++ {
// 		pbField := pbVal.Field(i)
// 		fieldName := pbVal.Type().Field(i).Name

// 		modelField := modelVal.FieldByName(fieldName)
// 		if modelField.IsValid() && modelField.CanSet() {
// 			modelField.Set(pbField)
// 		}
// 	}
// 	return &modelTeacher
// }

func GetTeachersFromDB(ctx context.Context, sortOptions bson.D, filter bson.M) ([]*pb.Teacher, error) {
	mongoClient, err := CreateMongoClient()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Unable to connect to mongodb.")
	}
	defer mongoClient.Disconnect(ctx)

	collection := mongoClient.Database("school").Collection("teachers")

	var cursor *mongo.Cursor
	if len(sortOptions) > 0 {
		cursor, err = collection.Find(ctx, filter, options.Find().SetSort(sortOptions))
	} else {
		cursor, err = collection.Find(ctx, filter)
	}
	if err != nil {
		return nil, utils.ErrorHandler(err, "Internal Server Error.")
	}
	defer cursor.Close(ctx)

	teachers, err := DecodeEntities(ctx, cursor, func() *models.Teacher { return &models.Teacher{} }, func() *pb.Teacher { return &pb.Teacher{} })
	if err != nil {
		return nil, err
	}
	return teachers, nil
}

// func DecodeTeachers(ctx context.Context, cursor *mongo.Cursor) ([]*pb.Teacher, error) {
// 	var teachers []*pb.Teacher
// 	for cursor.Next(ctx) {
// 		var modelTeacher models.Teacher
// 		var pbTeacher pb.Teacher
// 		err := cursor.Decode(&modelTeacher)
// 		if err != nil {
// 			return nil, utils.ErrorHandler(err, "Internal Server Error.")
// 		}
// 		//using reflect
// 		modelVal := reflect.ValueOf(&modelTeacher).Elem()
// 		modelType := modelVal.Type()

// 		pbVal := reflect.ValueOf(&pbTeacher).Elem()
// 		// pbType := pbVal.Type()

// 		for i := 0; i < modelVal.NumField(); i++ {
// 			field := modelVal.Field(i)
// 			fieldName := modelType.Field(i).Name

// 			if field.IsValid() && !field.IsZero() {
// 				pbField := pbVal.FieldByName(fieldName)
// 				if pbField.IsValid() && pbField.CanSet() {
// 					pbField.Set(field)
// 				}
// 			}
// 		}
// 		teachers = append(teachers, &pbTeacher)
// 	}
// 	return teachers, nil
// }

func UpdateTeachersInDB(ctx context.Context, teachers []*pb.Teacher) ([]*pb.Teacher, error) {
	mongoClient, err := CreateMongoClient()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Unable to connect to mongo.")
	}
	defer mongoClient.Disconnect(ctx)

	var updatedTeachers []*pb.Teacher
	for _, teacher := range teachers {
		if teacher.Id == "" {
			return nil, utils.ErrorHandler(fmt.Errorf("Id should not be empty"), "Id should not be empty")
		}
		// modelTeacher := MapProtoBufTeacherToModelTeacher(teacher)
		modelTeacher := MapProtoBufEntityToModelEntity(teacher, func() *models.Teacher { return &models.Teacher{} })

		objectId, err := primitive.ObjectIDFromHex(modelTeacher.Id)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Internal server error.")
		}

		bytesDoc, err := bson.Marshal(modelTeacher)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Internal Server Error")
		}

		var updatedDoc bson.M
		err = bson.Unmarshal(bytesDoc, &updatedDoc)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Internal server error.")
		}
		delete(updatedDoc, "_id")
		_, err = mongoClient.Database("school").Collection("teachers").UpdateOne(ctx, bson.M{"_id": objectId}, bson.M{"$set": updatedDoc})
		if err != nil {
			return nil, utils.ErrorHandler(err, fmt.Sprintf("Error updating teacher with id: %s", modelTeacher.Id))
		}

		// pbTeacher := mongodb.MapModelTeacherToProtoBufTeacher(modelTeacher)
		updatedTeachers = append(updatedTeachers, teacher)
	}
	return updatedTeachers, nil
}

func DeleteTeachersFromDB(ctx context.Context, teacherIdsToDelete []string) error {
	mongoClient, err := CreateMongoClient()
	if err != nil {
		return utils.ErrorHandler(err, "unable to connect to mongodb.")
	}
	defer mongoClient.Disconnect(ctx)
	// for _, id := range teacherIdsToDelete {
	// 	if id == "" {
	// 		return nil, utils.ErrorHandler(fmt.Errorf("id field should not be empty."), "id field should not be empty.")
	// 	}
	// 	objectId, err := primitive.ObjectIDFromHex(id)
	// 	if err != nil {
	// 		return nil, utils.ErrorHandler(err, "Internal server error.")
	// 	}

	// 	_, err = mongoClient.Database("school").Collection("teachers").DeleteOne(ctx, bson.M{"_id": objectId})
	// 	if err != nil {
	// 		return nil, utils.ErrorHandler(err, fmt.Sprintf("Error in deleting Teacher with id: %s", id))
	// 	}
	// 	deletedIds = append(deletedIds, id)
	// }

	objectIds := make([]primitive.ObjectID, len(teacherIdsToDelete))
	for i, id := range teacherIdsToDelete {
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
	fmt.Println(filter)

	result, err := mongoClient.Database("school").Collection("teachers").DeleteMany(ctx, filter)
	if err != nil {
		return utils.ErrorHandler(err, "Error in deleting teachers.")
	}

	if result.DeletedCount == 0 {
		return utils.ErrorHandler(fmt.Errorf("No Teachers are deleted."), "No Teachers are deleted.")
	}
	return nil
}

func GetStudentsByClassTeacherFromDB(ctx context.Context, id string) ([]*pb.Student, error) {
	if id == "" {
		return nil, utils.ErrorHandler(fmt.Errorf("id should not be empty"), "id should not be empty")
	}

	mongoClient, err := CreateMongoClient()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Unable to connect to mongodb.")
	}
	defer mongoClient.Disconnect(ctx)

	//convert string teacher Id to objectId
	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, utils.ErrorHandler(fmt.Errorf("Invalid id in request"), "Invalid id in request")
	}

	//Get the teacher with Id from request
	var teacher models.Teacher
	result := mongoClient.Database("school").Collection("teachers").FindOne(ctx, bson.M{"_id": objectId})
	err = result.Err()
	if err == mongo.ErrNoDocuments {
		return nil, utils.ErrorHandler(fmt.Errorf("Teacher not found"), "Teacher not found")
	}
	err = result.Decode(&teacher)
	if err != nil {
		return nil, utils.ErrorHandler(err, "Internal server Error")
	}

	//Get the students using teacher.Class
	var cursor *mongo.Cursor
	cursor, err = mongoClient.Database("school").Collection("students").Find(ctx, bson.M{"class": teacher.Class})
	if err != nil {
		return nil, utils.ErrorHandler(err, "Internal Server Error")
	}
	defer cursor.Close(ctx)

	pbStudents, err := DecodeEntities(ctx, cursor, func() *models.Student { return &models.Student{} }, func() *pb.Student { return &pb.Student{} })
	if err != nil {
		return nil, utils.ErrorHandler(err, "Internal Server Error")
	}
	return pbStudents, nil
}

func GetStudentsCountByClassTeacherFromDB(ctx context.Context, id string) (int64, error) {
	if id == "" {
		return 0, utils.ErrorHandler(fmt.Errorf("id field is empty in request."), "id field is empty in request.")
	}
	mongoClient, err := CreateMongoClient()
	if err != nil {
		return 0, utils.ErrorHandler(err, "Unable to connect to mongo")
	}

	//Get teacher details from db using ID from request
	var teacher models.Teacher
	objectId, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return 0, utils.ErrorHandler(fmt.Errorf("Invalid id in request"), "Invalid id in request")
	}

	result := mongoClient.Database("school").Collection("teachers").FindOne(ctx, bson.M{"_id": objectId})
	if result.Err() == mongo.ErrNoDocuments {
		return 0, utils.ErrorHandler(fmt.Errorf("Teacher not found"), "Teacher not found")
	}

	err = result.Decode(&teacher)
	if err != nil {
		return 0, utils.ErrorHandler(err, "Internal server error.")
	}

	//Get the count of students from the teacher's class
	noOfStudents, err := mongoClient.Database("school").Collection("students").CountDocuments(ctx, bson.M{"class": teacher.Class})
	if err != nil {
		return 0, utils.ErrorHandler(err, "Internal server error")
	}
	return noOfStudents, nil
}
