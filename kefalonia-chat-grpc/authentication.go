package main

import (
	"context"
	"fmt"
	protos "kefalonia-chat-grpc/proto"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserDb struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Name     string             `bson:"name"`
	Password string             `bson:"password"`
}

type AuthenticationServer struct {
	protos.UnimplementedAuthenticationServer

	authDb *mongo.Collection
}

func (s *AuthenticationServer) Register(ctx context.Context, in *protos.User) (*protos.RegisterReply, error) {
	//Check if the user already exists
	userCount, err := s.authDb.CountDocuments(context.Background(), bson.M{"name": in.Name})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Can't access database: %v", err))
	}
	if userCount > 0 {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("This user already exists: %v", in.Name))
	}

	// Create new user
	hashed, err := bcrypt.GenerateFromPassword([]byte(in.GetPassword()), bcrypt.DefaultCost)

	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Error while storing password: %v", err))
	}

	user := UserDb{
		Name:     in.Name,
		Password: string(hashed),
	}

	res, err := s.authDb.InsertOne(context.Background(), user)

	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("Error while storing password: %v", err))
	}

	oid := res.InsertedID.(primitive.ObjectID).Hex()

	return &protos.RegisterReply{
		Id: oid,
	}, nil
}

func (s *AuthenticationServer) Login(ctx context.Context, in *protos.User) (*protos.LoginReply, error) {
	log.Printf("Received: %v", in)
	return &protos.LoginReply{
		Token: "token",
	}, nil
}

func (s *AuthenticationServer) GetUsers(in *protos.Void, stream protos.Authentication_GetUsersServer) error {
	res, err := s.authDb.Find(context.Background(), bson.M{})

	if err != nil {
		return status.Errorf(codes.Internal, fmt.Sprintf("Can't fetch from Datab: %v", err))
	}

	data := &UserDb{}

	defer res.Close(context.Background())

	for res.Next(context.Background()) {
		err := res.Decode(data)

		if err != nil {
			return status.Errorf(codes.NotFound, fmt.Sprintf("Unable to get users: %v", err))
		}

		log.Printf("Received: %v", data)
		stream.Send(&protos.UserItem{
			Name: data.Name,
		})

	}
	if err := res.Err(); err != nil {
		return status.Errorf(codes.Internal, fmt.Sprintf("Unknown Database error: %v", err))
	}
	return nil
}
