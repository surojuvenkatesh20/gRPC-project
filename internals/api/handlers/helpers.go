package handlers

import (
	"grpcmongoproject/pkg/utils"
	"reflect"
	"strings"

	pb "grpcmongoproject/proto/gen"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func buildFilter(entity interface{}, entityModel interface{}) (bson.M, error) {
	var filter = bson.M{}
	if entity == nil || reflect.ValueOf(entity).IsNil() {
		return filter, nil
	}

	// var modelTeacher models.Teacher
	modelVal := reflect.ValueOf(entityModel).Elem()
	modelType := modelVal.Type()

	reqVal := reflect.ValueOf(entity).Elem()
	reqType := reqVal.Type()

	//Setting protobuf teacher values to models.Teacher variable
	for i := 0; i < reqVal.NumField(); i++ {
		field := reqVal.Field(i)
		fieldName := reqType.Field(i).Name

		if field.IsValid() && !field.IsZero() {
			modelField := modelVal.FieldByName(fieldName)
			if modelField.IsValid() && modelField.CanSet() {
				modelField.Set(field)
			}

		}
	}

	//Settings models.Teacher value to bson tags
	for i := 0; i < modelVal.NumField(); i++ {
		field := modelVal.Field(i)
		// fieldName := modelType.Field(i).Name

		if field.IsValid() && !field.IsZero() {
			bsonTag := modelType.Field(i).Tag.Get("bson")
			bsonTag = strings.TrimSuffix(bsonTag, ",omitempty")

			if bsonTag == "_id" {
				objId, err := primitive.ObjectIDFromHex(field.String())
				// objId, err := primitive.ObjectIDFromHex(entity)
				if err != nil {
					return nil, utils.ErrorHandler(err, "Invalid ID in request.")
				}
				filter[bsonTag] = objId
			} else {
				filter[bsonTag] = field.Interface().(string)
			}
		}
	}
	return filter, nil
}

func createSortFields(sortFields []*pb.SortField) bson.D {
	sortArray := bson.D{}

	for _, sortField := range sortFields {
		order := 1
		if sortField.OrderBy == pb.Order_DESC {
			order = -1
		}
		sortArray = append(sortArray, bson.E{Key: sortField.Field, Value: order})
	}

	return sortArray
}
