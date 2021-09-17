package handler

import (
	"context"

	"0chain.net/core/logging"
	"0chain.net/miner/minergrpc"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

func registerGRPCServices(r *mux.Router, server *grpc.Server) {
	minerService := newMinerGRPCService()
	grpcGatewayHandler := runtime.NewServeMux()

	minergrpc.RegisterMinerServiceServer(server, minerService)
	err := minergrpc.RegisterMinerServiceHandlerServer(context.Background(), grpcGatewayHandler, minerService)
	if err != nil {
		logging.Logger.Error("Error registering miner service handler" + err.Error())
		return
	}

	r.PathPrefix("/").Handler(grpcGatewayHandler)

	// add custom HandlePath
	//err = grpcGatewayHandler.HandlePath("POST", "/v1/file/upload/{allocation}",
	//	func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	//		r = mux.SetURLVars(r, map[string]string{"allocation": pathParams[`allocation`]})
	//		common.UserRateLimit(common.ToJSONResponse(WithConnection(UploadHandler)))(w, r)
	//	})
	//if err != nil {
	//	logging.Logger.Error("Error registering upload POST handler" + err.Error())
	//	return
	//}
}