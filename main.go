package main

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
)

func main() {
	iamSvc, ec2Svc := createSvc()

	createRoleOutput, err := createRole(iamSvc)

	if err != nil {
		log.Fatal(err)
	}

	createInstanceProfileOutput, err := createInstanceProfile(iamSvc, createRoleOutput)

	if err != nil {
		deleteRole(iamSvc, createRoleOutput)
		log.Fatal(err)
	}

	err = addRoleToInstanceProfile(iamSvc, createInstanceProfileOutput, createRoleOutput)

	if err != nil {
		deleteRole(iamSvc, createRoleOutput)
		deleteInstanceProfile(iamSvc, createInstanceProfileOutput)
		log.Fatal(err)
	}

	err = runInstances(ec2Svc, createInstanceProfileOutput)

	if err != nil {
		rollbackAll(iamSvc, createInstanceProfileOutput, createRoleOutput)
		log.Fatal(err)
	}

	rollbackAll(iamSvc, createInstanceProfileOutput, createRoleOutput)
}

func rollbackAll(iamSvc *iam.IAM, createInstanceProfileOutput *iam.CreateInstanceProfileOutput, createRoleOutput *iam.CreateRoleOutput) {
	removeProfile(iamSvc, createInstanceProfileOutput, createRoleOutput)
	deleteRole(iamSvc, createRoleOutput)
	deleteInstanceProfile(iamSvc, createInstanceProfileOutput)
}

func createSvc() (*iam.IAM, *ec2.EC2) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	iamSvc := iam.New(sess)
	ec2Svc := ec2.New(sess)
	return iamSvc, ec2Svc
}

func runInstances(ec2Svc *ec2.EC2, createInstanceProfileOutput *iam.CreateInstanceProfileOutput) error {
	_, err := ec2Svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:      aws.String("ami-0992fc94ca0f1415a"),
		DryRun:       aws.Bool(true),
		InstanceType: aws.String("t2.micro"),
		MaxCount:     aws.Int64(1),
		MinCount:     aws.Int64(1),
		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Arn: createInstanceProfileOutput.InstanceProfile.Arn,
		},
	})
	return err
}

func addRoleToInstanceProfile(iamSvc *iam.IAM, createInstanceProfileOutput *iam.CreateInstanceProfileOutput, createRoleOutput *iam.CreateRoleOutput) error {
	_, err := iamSvc.AddRoleToInstanceProfile(&iam.AddRoleToInstanceProfileInput{
		InstanceProfileName: createInstanceProfileOutput.InstanceProfile.InstanceProfileName,
		RoleName:            createRoleOutput.Role.RoleName,
	})
	return err
}

func removeProfile(iamSvc *iam.IAM, createInstanceProfileOutput *iam.CreateInstanceProfileOutput, createRoleOutput *iam.CreateRoleOutput) {
	iamSvc.RemoveRoleFromInstanceProfile(&iam.RemoveRoleFromInstanceProfileInput{
		InstanceProfileName: createInstanceProfileOutput.InstanceProfile.InstanceProfileName,
		RoleName:            createRoleOutput.Role.RoleName,
	})
}

func deleteInstanceProfile(iamSvc *iam.IAM, createInstanceProfileOutput *iam.CreateInstanceProfileOutput) {
	iamSvc.DeleteInstanceProfile(&iam.DeleteInstanceProfileInput{
		InstanceProfileName: createInstanceProfileOutput.InstanceProfile.InstanceProfileName,
	})
}

func createInstanceProfile(iamSvc *iam.IAM, createRoleOutput *iam.CreateRoleOutput) (*iam.CreateInstanceProfileOutput, error) {
	createInstanceProfileOutput, err := iamSvc.CreateInstanceProfile(&iam.CreateInstanceProfileInput{
		InstanceProfileName: createRoleOutput.Role.RoleName,
	})
	return createInstanceProfileOutput, err
}

func createRole(iamSvc *iam.IAM) (*iam.CreateRoleOutput, error) {
	createRoleOutput, err := iamSvc.CreateRole(&iam.CreateRoleInput{
		RoleName: aws.String("aws-role-test"),
		AssumeRolePolicyDocument: aws.String(
			`{
			   "Version" : "2012-10-17",
			   "Statement": [
			 	{
			 	  "Effect": "Allow",
			 	  "Principal": {
			 		"Service": [ "ec2.amazonaws.com" ]
			 	  },
			 	  "Action": "sts:AssumeRole"
			 	}
			   ]
			}`,
		),
	})
	return createRoleOutput, err
}

func deleteRole(iamSvc *iam.IAM, createRoleOutput *iam.CreateRoleOutput) {
	iamSvc.DeleteRole(&iam.DeleteRoleInput{
		RoleName: createRoleOutput.Role.RoleName,
	})
}
