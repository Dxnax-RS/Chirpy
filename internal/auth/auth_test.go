package auth

import(
	"testing"
	"time"
	"fmt"
	"github.com/google/uuid"
)

func TestMakeJWT(t *testing.T){
	duration := 24 * time.Hour
	//func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error)
	tests := []struct{
		userID 		uuid.UUID
		tokenSecret string
		expiresIn 	time.Duration
		wantErr 	bool
	}{
		{uuid.New(), "f12c1cf00a758bfc5df904632a68da06fb79b44e2730bc6ea58791f1637c0226", duration, false},
	}

	for _, tt := range tests {
		val, err := MakeJWT(tt.userID, tt.tokenSecret, tt.expiresIn)
		
		fmt.Printf("uuid is: %v", tt.userID)
		fmt.Printf("\nval is: %v\n", val)
		if (err != nil) != tt.wantErr{
			t.Errorf("MakeJWT() error = %v, wantErr = %v", err, tt.wantErr)
		}

	}
}

func TestValidateJWT(t *testing.T){
	//func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error)
	tests := []struct{
		tokenString string
		tokenSecret string
		wantErr 	bool
	}{
		{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJjaGlycHktYWNjZXNzIiwic3ViIjoiYzc2N2VhMTktZDViNi00NTE2LTk4NzItOWJhZTAzMGM3NjljIiwiZXhwIjoxNzc4MDkzNjA4LCJpYXQiOjE3NzgwMDcyMDh9.fy86zM3gWNFJUKUetELSjoIS42XbG5pHb2cRoCRjFYI", "f12c1cf00a758bfc5df904632a68da06fb79b44e2730bc6ea58791f1637c0226", false},
	}

	for _, tt := range tests {
		val, err := ValidateJWT(tt.tokenString, tt.tokenSecret)
		
		fmt.Printf("val is: %v\n", val)
		if (err != nil) != tt.wantErr{
			t.Errorf("MakeJWT() error = %v, wantErr = %v", err, tt.wantErr)
		}

	}
}