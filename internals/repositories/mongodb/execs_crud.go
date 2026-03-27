package mongodb

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"grpcmongoproject/internals/models"
	"grpcmongoproject/pkg/utils"
	"os"
	"strconv"
	"time"

	pb "grpcmongoproject/proto/gen"

	"github.com/go-mail/mail"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func AddExecsToDB(ctx context.Context, execsFromReq []*pb.Exec) ([]*pb.Exec, error) {
	mongoClient, err := CreateMongoClient()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Unable to connect to mongodb.")
	}
	defer mongoClient.Disconnect(ctx)

	//converting each protobuf field into models.Exec(bson) field
	newExecs := make([]*models.Exec, len(execsFromReq))
	for i, pbExec := range execsFromReq {
		newExecs[i] = MapProtoBufEntityToModelEntity(pbExec, func() *models.Exec { return &models.Exec{} })
		newExecs[i].Password, err = utils.EncodePassword(newExecs[i].Password)
		if err != nil {
			return nil, utils.ErrorHandler(fmt.Errorf("Error in Hashing password"), "Error in Hashing password")
		}
		currentTime := time.Now().Format(time.RFC3339)
		newExecs[i].UserCreatedAt = currentTime
		newExecs[i].InactiveStatus = false
	}

	//converting each models.Exec(bson) field into protobuf fields.
	addedExecs := []*pb.Exec{}
	for _, exec := range newExecs {
		result, err := mongoClient.Database("school").Collection("execs").InsertOne(ctx, exec)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Unable to create execs.")
		}

		objectId, ok := result.InsertedID.(primitive.ObjectID)
		if ok {
			exec.Id = objectId.Hex()
		}
		addedExecs = append(addedExecs, MapModelEntityToProtoBufEntity(exec, func() *pb.Exec { return &pb.Exec{} }))

	}
	return addedExecs, nil
}

func GetExecsFromDB(ctx context.Context, sortOptions bson.D, filter bson.M, pageNumber, pageSize uint32) ([]*pb.Exec, error) {
	mongoClient, err := CreateMongoClient()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Unable to connect to mongodb.")
	}
	defer mongoClient.Disconnect(ctx)

	collection := mongoClient.Database("school").Collection("execs")

	if pageNumber < 1 {
		pageNumber = 1
	}
	if pageSize < 10 {
		pageSize = 10
	}
	findOptions := options.Find()
	findOptions.SetSkip(int64((pageNumber - 1) * pageSize))
	findOptions.SetLimit(int64(pageSize))
	findOptions.SetSort(sortOptions)

	var cursor *mongo.Cursor
	if len(sortOptions) > 0 {
		cursor, err = collection.Find(ctx, filter, findOptions)
	} else {
		cursor, err = collection.Find(ctx, filter)
	}
	if err != nil {
		return nil, utils.ErrorHandler(err, "Internal Server Error.")
	}

	execs, err := DecodeEntities(ctx, cursor, func() *models.Exec { return &models.Exec{} }, func() *pb.Exec { return &pb.Exec{} })
	if err != nil {
		return nil, err
	}
	return execs, nil
}

func UpdateExecsInDB(ctx context.Context, execs []*pb.Exec) ([]*pb.Exec, error) {
	mongoClient, err := CreateMongoClient()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Unable to connect to mongo.")
	}
	defer mongoClient.Disconnect(ctx)

	var updatedExecs []*pb.Exec
	for _, exec := range execs {
		if exec.Id == "" {
			return nil, utils.ErrorHandler(fmt.Errorf("Id should not be empty"), "Id should not be empty")
		}
		modelExec := MapProtoBufEntityToModelEntity(exec, func() *models.Exec { return &models.Exec{} })

		objectId, err := primitive.ObjectIDFromHex(modelExec.Id)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Invalid id in request.")
		}

		bytesDoc, err := bson.Marshal(modelExec)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Internal Server Error")
		}

		var updatedDoc bson.M
		err = bson.Unmarshal(bytesDoc, &updatedDoc)
		if err != nil {
			return nil, utils.ErrorHandler(err, "Internal server error.")
		}
		delete(updatedDoc, "_id")
		_, err = mongoClient.Database("school").Collection("execs").UpdateOne(ctx, bson.M{"_id": objectId}, bson.M{"$set": updatedDoc})
		if err != nil {
			return nil, utils.ErrorHandler(err, fmt.Sprintf("Error updating exec with id: %s", modelExec.Id))
		}

		updatedExecs = append(updatedExecs, exec)
	}
	return updatedExecs, nil
}

func DeleteExecsFromDB(ctx context.Context, execIdsToDelete []string) error {
	mongoClient, err := CreateMongoClient()
	if err != nil {
		return utils.ErrorHandler(err, "unable to connect to mongodb.")
	}
	defer mongoClient.Disconnect(ctx)

	objectIds := make([]primitive.ObjectID, len(execIdsToDelete))
	for i, id := range execIdsToDelete {
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

	result, err := mongoClient.Database("school").Collection("execs").DeleteMany(ctx, filter)
	if err != nil {
		return utils.ErrorHandler(err, "Error in deleting execs.")
	}

	if result.DeletedCount == 0 {
		return utils.ErrorHandler(fmt.Errorf("No Execs are deleted."), "No Execs are deleted.")
	}
	return nil
}

func GetExecByUsername(ctx context.Context, username string) (*models.Exec, error) {
	mongoClient, err := CreateMongoClient()
	if err != nil {
		return nil, utils.ErrorHandler(err, "Unable to connect to mongo.")
	}
	defer mongoClient.Disconnect(ctx)

	//Check if any user is present with the entered username
	var exec models.Exec
	result := mongoClient.Database("school").Collection("execs").FindOne(ctx, bson.M{"username": username})
	if result.Err() == mongo.ErrNoDocuments {
		fmt.Println(result.Err())
		return nil, utils.ErrorHandler(result.Err(), "Username does not exists.")
	}
	err = result.Decode(&exec)
	if err != nil {
		return nil, utils.ErrorHandler(err, "Internal server error.")
	}

	return &exec, nil
}

func UpdateExecPasswordInDB(ctx context.Context, req *pb.ExecUpdatePasswordRequest) (string, string, error) {
	//Get exec based on ID
	//Check if current password in request == hashed password, else throw err
	//if current password verified, hash the new password and update in db
	//Generate a new JWT token
	if req.Id == "" || req.CurrentPassword == "" || req.NewPassword == "" {
		return "", "", utils.ErrorHandler(fmt.Errorf("Exec id, current password and new password are required."), "Exec id, current password and new password are required.")
	}

	mongoClient, err := CreateMongoClient()
	if err != nil {
		return "", "", utils.ErrorHandler(err, "Unable to connect to mongo.")
	}
	defer mongoClient.Disconnect(ctx)

	//Get the exec from DB using if from request
	var exec models.Exec
	objectId, err := primitive.ObjectIDFromHex(req.Id)
	if err != nil {
		return "", "", utils.ErrorHandler(fmt.Errorf("Invalid id in request."), "Invalid id in request")
	}
	result := mongoClient.Database("school").Collection("execs").FindOne(ctx, bson.M{"_id": objectId})
	if result.Err() == mongo.ErrNoDocuments {
		return "", "", utils.ErrorHandler(fmt.Errorf("User does not exists."), "User does not exists.")
	}
	err = result.Decode(&exec)
	if err != nil {
		return "", "", utils.ErrorHandler(err, "Internal server error.")
	}

	//Verify if current password hash is matching with DB
	err = utils.VerifyPassword(req.CurrentPassword, exec.Password)
	if err != nil {
		return "", "", utils.ErrorHandler(fmt.Errorf("current password is incorrect"), "current password is incorrect")
	}

	//Hash the new password from request and update the record in db
	exec.Password, err = utils.EncodePassword(req.NewPassword)
	if err != nil {
		return "", "", utils.ErrorHandler(err, "Internal Server Error")
	}
	currentTime := time.Now().Format(time.RFC3339)
	exec.PasswordChangedAt = currentTime

	updates := bson.M{
		"$set": bson.M{
			"password":            exec.Password,
			"password_changed_at": exec.PasswordChangedAt,
		},
	}

	_, err = mongoClient.Database("school").Collection("execs").UpdateOne(ctx, bson.M{"_id": objectId}, updates)
	if err != nil {
		return "", "", utils.ErrorHandler(err, "Error in updating Pasword.")
	}
	return exec.Username, exec.Role, nil
}

func ExecsDeactivateInDB(ctx context.Context, execIds *pb.ExecIds) (bool, error) {
	//for each ID => generate one objectId
	//use UpdateMany() to update status in one single go
	if len(execIds.Ids) == 0 {
		return false, utils.ErrorHandler(fmt.Errorf("ids should not be empty"), "ids should not be empty")
	}

	objectIds := []primitive.ObjectID{}
	for _, execId := range execIds.Ids {
		objectId, err := primitive.ObjectIDFromHex(execId.Id)
		if err != nil {
			return false, utils.ErrorHandler(err, "Invalid id in request.")
		}
		objectIds = append(objectIds, objectId)
	}

	mongoClient, err := CreateMongoClient()
	if err != nil {
		return false, utils.ErrorHandler(err, "Unable to connect to mongo.")
	}
	defer mongoClient.Disconnect(ctx)

	filter := bson.M{
		"_id": bson.M{
			"$in": objectIds,
		},
	}
	updates := bson.M{
		"$set": bson.M{
			"inactive_status": true,
		},
	}
	_, err = mongoClient.Database("school").Collection("execs").UpdateMany(ctx, filter, updates)
	if err != nil {
		return false, utils.ErrorHandler(err, fmt.Sprintf("Error in updating Db records: %v", err))
	}
	return true, nil
}

func ForgotPasswordDB(ctx context.Context, email string) (string, error) {
	//Check if any user exists with the email
	//If user exists, generate a hash token that is active for next 10 minutes
	mongoClient, err := CreateMongoClient()
	if err != nil {
		return "", utils.ErrorHandler(err, "Unable to connect to mongodb.")
	}

	var exec models.Exec
	result := mongoClient.Database("school").Collection("execs").FindOne(ctx, bson.M{"email": email})
	if result.Err() == mongo.ErrNoDocuments {
		return "", utils.ErrorHandler(fmt.Errorf("User does not exists."), ("User does not exists."))
	}

	err = result.Decode(&exec)
	if err != nil {
		return "", utils.ErrorHandler(err, "Internal server error.")
	}

	tokenBytes := make([]byte, 16)
	_, err = rand.Read(tokenBytes)
	if err != nil {
		return "", utils.ErrorHandler(nil, "Internal server error")
	}

	token := hex.EncodeToString(tokenBytes)
	hashedToken := sha256.Sum256(tokenBytes)
	tokenString := hex.EncodeToString(hashedToken[:])

	duration, err := strconv.Atoi(os.Getenv("PASSWORD_RESET_TOKEN_TIME"))
	if err != nil {
		return "", utils.ErrorHandler(err, "Internal server error.")
	}

	mins := time.Duration(duration)
	resetTime := time.Now().Add(mins * time.Minute).Format(time.RFC3339)

	updates := bson.M{
		"$set": bson.M{
			"password_token_expires": resetTime,
			"password_reset_token":   tokenString,
		},
	}
	_, err = mongoClient.Database("school").Collection("execs").UpdateOne(ctx, bson.M{"email": email}, updates)
	fmt.Println("Updated token, and time")
	if err != nil {
		return "", utils.ErrorHandler(err, "Internal Server Error")
	}

	url := fmt.Sprintf("https://localhost:50051/resetpassword/reset/%s", token)
	message := fmt.Sprintf("Reset your password with the below link.\n%s\n. Please us reset code: %s along with reset URL. Link will be active only for %v minutes.", url, token, mins)
	subject := "Password Reset Email!"

	m := mail.NewMessage()
	m.SetHeader("From", "schoolAcademy@ac.in")
	m.SetHeader("To", email)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", message)

	d := mail.NewDialer("localhost", 1025, "", "")
	err = d.DialAndSend(m)
	if err != nil {
		cleanup := bson.M{
			"$set": bson.M{
				"password_token_expires": "",
				"password_reset_token":   "",
			},
		}
		_, err = mongoClient.Database("school").Collection("execs").UpdateOne(ctx, bson.M{"email": email}, cleanup)
		return "", utils.ErrorHandler(err, "Error in sending password reset email. Please try after some time.")
	}
	return message, nil
}

func ResetPasswordInDB(ctx context.Context, req *pb.ExecResetPasswordRequest) (bool, error) {
	//Get token string from Forgot password request
	//Decode, hash and encode.
	//Check this token is matching with user in DB and current time and less than token expires.
	//Update the password in db

	//Decode, Hash and Encode the token
	tokenBytes, err := hex.DecodeString(req.Token)
	if err != nil {
		return false, utils.ErrorHandler(err, "Internal server error.")
	}
	hashToken := sha256.Sum256(tokenBytes)
	encodedToken := hex.EncodeToString(hashToken[:])

	mongoClient, err := CreateMongoClient()
	if err != nil {
		return false, utils.ErrorHandler(err, "Unable to connect to mongo.")
	}

	var exec models.Exec
	filter := bson.M{
		"password_reset_token": encodedToken,
		"password_token_expires": bson.M{
			"$gte": time.Now().Format(time.RFC3339),
		},
	}
	result := mongoClient.Database("school").Collection("execs").FindOne(ctx, filter)
	if result.Err() == mongo.ErrNoDocuments {
		return false, utils.ErrorHandler(fmt.Errorf("Token invalid/expired. Please generate again."), "Token invalid/incorrect.")
	}
	err = result.Decode(&exec)
	if err != nil {
		return false, utils.ErrorHandler(err, "Internal server error.")
	}
	//If token is currect and current time < password expires time, then update the exec with new password
	//Reset token and token expires fields to empty string
	exec.Password, err = utils.EncodePassword(req.NewPassword)
	if err != nil {
		return false, utils.ErrorHandler(err, "Internal server error.")
	}

	updates := bson.M{
		"$set": bson.M{
			"password":               exec.Password,
			"password_reset_token":   "",
			"password_token_expires": "",
			"password_changed_at":    time.Now().Format(time.RFC3339),
		},
	}

	_, err = mongoClient.Database("school").Collection("execs").UpdateOne(ctx, bson.M{"email": exec.Email}, updates)
	if err != nil {
		return false, utils.ErrorHandler(err, "Internal server error.")
	}
	return true, nil
}
