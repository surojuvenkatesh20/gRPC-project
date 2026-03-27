package mongodb

import (
	"context"
	"grpcmongoproject/pkg/utils"
	"reflect"

	"go.mongodb.org/mongo-driver/mongo"
)

// T is pb type (teacher, student, exec) and M is model type (teacher, student, exec)
func DecodeEntities[T any, M any](ctx context.Context, cursor *mongo.Cursor, newModel func() *M, newEntity func() *T) ([]*T, error) {
	var entities []*T
	for cursor.Next(ctx) {
		model := newModel()
		err := cursor.Decode(model)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Internal Server Error.")
		}
		//using reflect
		modelVal := reflect.ValueOf(model).Elem()
		modelType := modelVal.Type()

		entity := newEntity()
		pbVal := reflect.ValueOf(entity).Elem()
		// pbType := pbVal.Type()

		for i := 0; i < modelVal.NumField(); i++ {
			field := modelVal.Field(i)
			fieldName := modelType.Field(i).Name

			if field.IsValid() && !field.IsZero() {
				pbField := pbVal.FieldByName(fieldName)
				if pbField.IsValid() && pbField.CanSet() {
					pbField.Set(field)
				}
			}
		}
		entities = append(entities, entity)
	}

	err := cursor.Err()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Internal server error.")
	}
	return entities, nil
}

func MapModelEntityToProtoBufEntity[P, M any](modelEntity M, newProtoEntity func() *P) *P {
	protoEntity := newProtoEntity()
	modelVal := reflect.ValueOf(modelEntity).Elem()
	pbEntity := reflect.ValueOf(protoEntity).Elem()

	for i := 0; i < modelVal.NumField(); i++ {
		field := modelVal.Field(i)
		fieldName := modelVal.Type().Field(i).Name

		pbField := pbEntity.FieldByName(fieldName)

		if pbField.IsValid() && pbField.CanSet() {
			pbField.Set(field)
		}
	}
	return protoEntity
}

func MapProtoBufEntityToModelEntity[P, M any](protoEntity P, newModelEntity func() *M) *M {
	modelEntity := newModelEntity()
	pbVal := reflect.ValueOf(protoEntity).Elem()
	modelVal := reflect.ValueOf(modelEntity).Elem()

	for i := 0; i < pbVal.NumField(); i++ {
		pbField := pbVal.Field(i)
		fieldName := pbVal.Type().Field(i).Name

		modelField := modelVal.FieldByName(fieldName)
		if modelField.IsValid() && modelField.CanSet() {
			modelField.Set(pbField)
		}
	}
	return modelEntity
}
