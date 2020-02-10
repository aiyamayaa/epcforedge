/* SPDX-License-Identifier: Apache-2.0
* Copyright (c) 2019 Intel Corporation
 */

package ngcnef

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"

	//"strconv"

	"github.com/gorilla/mux"
)

func createNewPFDTrans(nefCtx *nefContext, afID string,
	trans PfdManagement) (loc string, rsp nefSBRspData, err error) {

	var af *afData
	nef := &nefCtx.nef

	af, err = nef.nefGetAf(afID)

	if err != nil {
		log.Err("NO AF PRESENT CREATE AF")
		af, err = nef.nefAddAf(nefCtx, afID)
		if err != nil {
			return loc, rsp, err
		}
	} else {
		log.Infoln("AF PRESENT")
	}

	loc, rsp, err = af.afAddPFDTransaction(nefCtx, trans)

	if err != nil {
		return loc, rsp, err
	}

	return loc, rsp, nil
}

// ReadAllPFDManagementTransaction : API to read all the PFD Transactions
func ReadAllPFDManagementTransaction(w http.ResponseWriter,
	r *http.Request) {

	var pfdTrans []PfdManagement
	var rsp nefSBRspData
	var err error

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID : %s", vars["scsAsId"])

	af, err := nef.nefGetAf(vars["scsAsId"])

	if err != nil {
		/* Failure in getting AF with afId received. In this case no
		 * transaction data will be returned to AF */
		log.Infoln(err)
	} else {
		rsp, pfdTrans, err = af.afGetPfdTransactionList(nefCtx)
		if err != nil {
			log.Err(err)
			sendErrorResponseToAF(w, rsp)
			return
		}
	}

	mdata, err2 := json.Marshal(pfdTrans)

	if err2 != nil {
		sendCustomeErrorRspToAF(w, 400, "Failed to MARSHAL Subscription data ")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	//Send Success response to Network
	_, err = w.Write(mdata)
	if err != nil {
		log.Errf("Write Failed: %v", err)
		return
	}

	log.Infof("HTTP Response sent: %d", http.StatusOK)
}

// CreatePFDManagementTransaction  Handles the PFD Management requested
// by AF
func CreatePFDManagementTransaction(w http.ResponseWriter,
	r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["scsAsId"])

	b, err := ioutil.ReadAll(r.Body)
	defer closeReqBody(r)

	if err != nil {
		sendCustomeErrorRspToAF(w, 400, "Failed to read HTTP POST Body")
		return
	}

	//Pfd Management data
	pfdBody := PfdManagement{}

	//Convert the json Traffic Influence data into struct
	err1 := json.Unmarshal(b, &pfdBody)

	if err1 != nil {
		log.Err(err1)
		sendCustomeErrorRspToAF(w, 400, "Failed UnMarshal POST data")
		return
	}
	pfdBody.PfdReports = make(map[string]PfdReport)
	//TBD Validate the params
	loc, rsp, err3 := createNewPFDTrans(nefCtx, vars["scsAsId"], pfdBody)

	if err3 != nil {
		log.Err(err3)

		if rsp.errorCode == 500 {
			send500PFDResponseToAF(w, rsp, pfdBody.PfdReports)

		} else {

			sendErrorResponseToAF(w, rsp)

		}
		return
	}
	log.Infoln(loc)

	pfdBody.Self = Link(loc)

	//Martshal data and send into the body
	mdata, err2 := json.Marshal(pfdBody)

	if err2 != nil {
		log.Err(err2)
		sendCustomeErrorRspToAF(w, 400, "Failed to Marshal GET response data")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Location", loc)

	// Response should be 201 Created as per 3GPP 29.522
	w.WriteHeader(http.StatusCreated)
	log.Infof("CreatePFDManagementresponses => %d",
		http.StatusCreated)
	_, err = w.Write(mdata)
	if err != nil {
		log.Errf("Write Failed: %v", err)
		return
	}
	nef := &nefCtx.nef
	logNef(nef)

}

// ReadPFDManagementTransaction : Read a particular PFD transaction details
func ReadPFDManagementTransaction(w http.ResponseWriter, r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["scsAsId"])
	log.Infof(" TRANSACTION ID  : %s", vars["transactionId"])

	af, ok := nef.nefGetAf(vars["scsAsId"])

	if ok != nil {
		sendCustomeErrorRspToAF(w, 404, "Failed to find AF records")
		return
	}

	rsp, pfdTrans, err := af.afGetPfdTransaction(nefCtx, vars["transactionId"])

	if err != nil {
		log.Err(err)
		sendErrorResponseToAF(w, rsp)
		return
	}

	mdata, err2 := json.Marshal(pfdTrans)
	if err2 != nil {
		log.Err(err2)
		sendCustomeErrorRspToAF(w, 400, "Failed to Marshal GETPFDresponse data")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	_, err = w.Write(mdata)
	if err != nil {
		log.Errf("Write Failed: %v", err)
		return
	}

	log.Infof("HTTP Response sent: %d", http.StatusOK)
}

// ReadPFDManagementApplication : Read a particular PFD transaction details of
// and external application identifier
func ReadPFDManagementApplication(w http.ResponseWriter, r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["scsAsId"])
	log.Infof(" PFD TRANSACTION ID  : %s", vars["transactionId"])
	log.Infof(" PFD APPLICATION ID : %s", vars["appId"])

	af, ok := nef.nefGetAf(vars["scsAsId"])

	if ok != nil {
		sendCustomeErrorRspToAF(w, 404, "Failed to find AF records")
		return
	}

	pfdTransID := vars["transactionId"]
	appID := vars["appId"]
	rsp, pfdData, err := af.afGetPfdApplication(nefCtx, pfdTransID, appID)

	if err != nil {
		log.Err(err)
		sendErrorResponseToAF(w, rsp)
		return
	}

	mdata, err2 := json.Marshal(pfdData)
	if err2 != nil {
		log.Err(err2)
		sendCustomeErrorRspToAF(w, 400, "Failed to Marshal GET response data")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	_, err = w.Write(mdata)
	if err != nil {
		log.Errf("Write Failed: %v", err)
		return
	}

	log.Infof("HTTP Response sent: %d", http.StatusOK)
}

// DeletePFDManagementApplication deletes the existing PFD transaction of
// an application identifier
func DeletePFDManagementApplication(w http.ResponseWriter,
	r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["scsAsId"])
	log.Infof(" PFD TRANSACTION ID  : %s", vars["transactionId"])
	log.Infof(" PFD APPLICATION ID  : %s", vars["appId"])

	af, err := nef.nefGetAf(vars["scsAsId"])

	if err != nil {
		log.Err(err)
		sendCustomeErrorRspToAF(w, 404, "Failed to find AF entry")
		return
	}

	pfdTransID := vars["transactionId"]
	appID := vars["appId"]

	rsp, err := af.afDeletePfdApplication(nefCtx, pfdTransID, appID)

	if err != nil {
		log.Err(err)
		sendErrorResponseToAF(w, rsp)
		return
	}
	// Response should be 204 as per 3GPP 29.522
	w.WriteHeader(http.StatusNoContent)

	log.Infof("HTTP Response sent: %d", http.StatusNoContent)

	logNef(nef)
}

// DeletePFDManagementTransaction deletes the existing PFD transaction
func DeletePFDManagementTransaction(w http.ResponseWriter,
	r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["scsAsId"])
	log.Infof(" PFD TRANSACTION ID  : %s", vars["transactionId"])

	af, err := nef.nefGetAf(vars["scsAsId"])

	if err != nil {
		log.Err(err)
		sendCustomeErrorRspToAF(w, 404, "Failed to find AF entry")
		return
	}
	rsp, err := af.afDeletePfdTransaction(nefCtx, vars["transactionId"])

	if err != nil {
		log.Err(err)
		sendErrorResponseToAF(w, rsp)
		return
	}

	// Response should be 204 as per 3GPP 29.522
	w.WriteHeader(http.StatusNoContent)

	log.Infof("HTTP Response sent: %d", http.StatusNoContent)

	// If the AF subcount and transaction count is 0 delete the AF

	logNef(nef)
}

// UpdatePutPFDManagementTransaction updates an existing PFD transaction
func UpdatePutPFDManagementTransaction(w http.ResponseWriter,
	r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["scsAsId"])
	log.Infof(" PFD TRANSACTION ID  : %s", vars["transactionId"])

	af, ok := nef.nefGetAf(vars["scsAsId"])
	if ok == nil {

		b, err := ioutil.ReadAll(r.Body)
		defer closeReqBody(r)

		if err != nil {
			log.Err(err)
			sendCustomeErrorRspToAF(w, 400, "Failed to read HTTP PUT Body")
			return
		}

		//PFD Transaction data
		pfdTrans := PfdManagement{}

		//Convert the json Traffic Influence data into struct
		err1 := json.Unmarshal(b, &pfdTrans)

		if err1 != nil {
			log.Err(err1)
			sendCustomeErrorRspToAF(w, 400, "Failed UnMarshal PUT data")
			return
		}

		rsp, newPfdTrans, err := af.afUpdatePutPfdTransaction(nefCtx,
			vars["transactionId"], pfdTrans)

		if err != nil {
			sendErrorResponseToAF(w, rsp)
			return
		}

		mdata, err2 := json.Marshal(newPfdTrans)

		if err2 != nil {
			log.Err(err2)
			sendCustomeErrorRspToAF(w, 400, "Failed to Marshal PUT"+
				"response data")
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")

		_, err = w.Write(mdata)
		if err != nil {
			log.Errf("Write Failed: %v", err)
		}
		return

	}
	log.Infoln(ok)
	sendCustomeErrorRspToAF(w, 404, "Failed to find AF records")

}

// UpdatePutPFDManagementApplication updates an existing PFD transaction
func UpdatePutPFDManagementApplication(w http.ResponseWriter,
	r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["scsAsId"])
	log.Infof(" PFD TRANSACTION ID  : %s", vars["transactionId"])
	log.Infof(" PFD APPLICATION ID  : %s", vars["appId"])

	af, ok := nef.nefGetAf(vars["scsAsId"])
	if ok == nil {

		b, err := ioutil.ReadAll(r.Body)
		defer closeReqBody(r)

		if err != nil {
			log.Err(err)
			sendCustomeErrorRspToAF(w, 400, "Failed to read HTTP PUT Body")
			return
		}

		//PFD Transaction data
		pfdData := PfdData{}

		//Convert the json PFD Management data into struct
		err1 := json.Unmarshal(b, &pfdData)

		if err1 != nil {
			log.Err(err1)
			sendCustomeErrorRspToAF(w, 400, "Failed UnMarshal PUT data")
			return
		}

		rsp, newPfdData, err := af.afUpdatePutPfdApplication(nefCtx,
			vars["transactionId"], vars["appId"], pfdData)

		if err != nil {
			sendErrorResponseToAF(w, rsp)
			return
		}

		mdata, err2 := json.Marshal(newPfdData)

		if err2 != nil {
			log.Err(err2)
			sendCustomeErrorRspToAF(w, 400, "Failed to Marshal PUT"+
				"response data")
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")

		_, err = w.Write(mdata)
		if err != nil {
			log.Errf("Write Failed: %v", err)
		}
		return

	}
	log.Infoln(ok)
	sendCustomeErrorRspToAF(w, 404, "Failed to find AF records")

}

// PatchPFDManagementApplication patches the PFD application PFDs
func PatchPFDManagementApplication(w http.ResponseWriter,
	r *http.Request) {

	nefCtx := r.Context().Value(nefCtxKey("nefCtx")).(*nefContext)
	nef := &nefCtx.nef

	vars := mux.Vars(r)
	log.Infof(" AFID  : %s", vars["scsAsId"])
	log.Infof(" PFD TRANSACTION ID  : %s", vars["transactionId"])
	log.Infof(" PFD APPLICATION ID  : %s", vars["appId"])

	af, ok := nef.nefGetAf(vars["scsAsId"])
	if ok == nil {

		b, err := ioutil.ReadAll(r.Body)
		defer closeReqBody(r)

		if err != nil {
			log.Err(err)
			sendCustomeErrorRspToAF(w, 400, "Failed to read HTTP PUT Body")
			return
		}

		//PFD Transaction data
		pfdData := PfdData{}

		//Convert the json PFD Management data into struct
		err1 := json.Unmarshal(b, &pfdData)

		if err1 != nil {
			log.Err(err1)
			sendCustomeErrorRspToAF(w, 400, "Failed UnMarshal PUT data")
			return
		}

		rsp, newPfdData, err := af.afUpdatePatchPfdApplication(nefCtx,
			vars["transactionId"], vars["appId"], pfdData)

		if err != nil {
			sendErrorResponseToAF(w, rsp)
			return
		}

		mdata, err2 := json.Marshal(newPfdData)

		if err2 != nil {
			log.Err(err2)
			sendCustomeErrorRspToAF(w, 400, "Failed to Marshal PUT"+
				"response data")
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")

		_, err = w.Write(mdata)
		if err != nil {
			log.Errf("Write Failed: %v", err)
		}
		return

	}
	log.Infoln(ok)
	sendCustomeErrorRspToAF(w, 404, "Failed to find AF records")

}

//PFD Management functions

func (af *afData) afUpdatePutPfdApplication(nefCtx *nefContext, transID string,
	appID string, pfdData PfdData) (rsp nefSBRspData, updPfd PfdData,
	err error) {

	pfdTrans, ok := af.pfdtrans[transID]

	if !ok {
		rsp.errorCode = 400
		rsp.pd.Title = pfdNotFound

		return rsp, updPfd, errors.New(pfdNotFound)
	}

	/*rsp, err = sub.NEFSBPut(sub, nefCtx, ti)

	if err != nil {
		log.Err("Failed to Update Subscription")
		return rsp, updtTI, err
	}*/

	trans, ok := pfdTrans.pfdManagement.PfdDatas[appID]

	if !ok {
		rsp.errorCode = 404
		rsp.pd.Title = appNotFound
		return rsp, trans, errors.New(appNotFound)
	}

	pfdTrans.pfdManagement.PfdDatas[appID] = pfdData
	updPfd = pfdData

	log.Infoln("Update PFD transaction Successful")
	return rsp, updPfd, err
}

func (af *afData) afUpdatePatchPfdApplication(nefCtx *nefContext,
	transID string, appID string, pfdData PfdData) (rsp nefSBRspData,
	updPfd PfdData, err error) {

	pfdTrans, ok := af.pfdtrans[transID]

	if !ok {
		rsp.errorCode = 400
		rsp.pd.Title = pfdNotFound

		return rsp, updPfd, errors.New(pfdNotFound)
	}

	/*rsp, err = sub.NEFSBPut(sub, nefCtx, ti)

	if err != nil {
		log.Err("Failed to Update Subscription")
		return rsp, updtTI, err
	}*/

	trans, ok := pfdTrans.pfdManagement.PfdDatas[appID]

	if !ok {
		rsp.errorCode = 404
		rsp.pd.Title = appNotFound
		return rsp, trans, errors.New(appNotFound)
	}
	// Updating the PFDs present in the
	for key := range pfdData.Pfds {

		pfd, ok := trans.Pfds[key]
		if ok {
			pfd = pfdData.Pfds[key]
			trans.Pfds[key] = pfd
			log.Infof("PFD id %s updated by PATCH ", pfd.PfdID)
		}

	}

	updPfd = trans
	log.Infoln("Patch PFD Application PFDs Successful")
	return rsp, updPfd, err
}

func (af *afData) afUpdatePutPfdTransaction(nefCtx *nefContext, transID string,
	trans PfdManagement) (rsp nefSBRspData, updPfd PfdManagement, err error) {

	pfdTrans, ok := af.pfdtrans[transID]

	if !ok {
		rsp.errorCode = 400
		rsp.pd.Title = pfdNotFound

		return rsp, updPfd, errors.New(pfdNotFound)
	}

	/*rsp, err = sub.NEFSBPut(sub, nefCtx, ti)

	if err != nil {
		log.Err("Failed to Update Subscription")
		return rsp, updtTI, err
	}*/

	updPfd = trans
	updPfd.Self = pfdTrans.pfdManagement.Self
	pfdTrans.pfdManagement = updPfd

	log.Infoln("Update PFD transaction Successful")
	return rsp, updPfd, err
}

func (af *afData) afDeletePfdTransaction(nefCtx *nefContext,
	pfdTrans string) (rsp nefSBRspData, err error) {

	//Check if PFD transaction is already present
	_, ok := af.pfdtrans[pfdTrans]

	if !ok {
		rsp.errorCode = 404
		rsp.pd.Title = pfdNotFound
		return rsp, errors.New(pfdNotFound)
	}

	/*rsp, err = sub.NEFSBDelete(sub, nefCtx)

	if err != nil {
		log.Err("Failed to Delete Subscription")
		return rsp, err
	}*/

	//Delete local entry in map of pfd transactions
	delete(af.pfdtrans, pfdTrans)

	return rsp, err
}

func (af *afData) afGetPfdApplication(nefCtx *nefContext,
	transID string, appID string) (rsp nefSBRspData, trans PfdData, err error) {

	transPfd, ok := af.pfdtrans[transID]

	if !ok {
		rsp.errorCode = 404
		rsp.pd.Title = pfdNotFound
		return rsp, trans, errors.New(pfdNotFound)
	}

	trans, ok = transPfd.pfdManagement.PfdDatas[appID]

	if !ok {
		rsp.errorCode = 404
		rsp.pd.Title = appNotFound
		return rsp, trans, errors.New(appNotFound)
	}

	//Return locally
	return rsp, trans, err
}

func (af *afData) afDeletePfdApplication(nefCtx *nefContext,
	transID string, appID string) (rsp nefSBRspData, err error) {

	transPfd, ok := af.pfdtrans[transID]

	if !ok {
		rsp.errorCode = 404
		rsp.pd.Title = pfdNotFound
		return rsp, errors.New(pfdNotFound)
	}

	_, ok = transPfd.pfdManagement.PfdDatas[appID]

	if !ok {
		rsp.errorCode = 404
		rsp.pd.Title = appNotFound
		return rsp, errors.New(appNotFound)
	}
	delete(transPfd.pfdManagement.PfdDatas, appID)
	//Return locally
	return rsp, err
}

func (af *afData) afGetPfdTransaction(nefCtx *nefContext,
	transID string) (rsp nefSBRspData, trans PfdManagement, err error) {

	transPfd, ok := af.pfdtrans[transID]

	if !ok {
		rsp.errorCode = 404
		rsp.pd.Title = pfdNotFound
		return rsp, trans, errors.New(pfdNotFound)
	}

	//ti, rsp, err = sub.NEFSBGet(sub, nefCtx)

	/*
		if err != nil {
			log.Infoln("Failed to Get Subscription")
			return rsp, ti, err
		}

		return rsp, ti, err
	*/

	//Return locally
	return rsp, transPfd.pfdManagement, err
}

func (af *afData) afGetPfdTransactionList(nefCtx *nefContext) (rsp nefSBRspData,
	transList []PfdManagement, err error) {

	var transPfd PfdManagement

	if len(af.pfdtrans) > 0 {

		for key := range af.pfdtrans {

			rsp, transPfd, err = af.afGetPfdTransaction(nefCtx, key)

			if err != nil {
				return rsp, transList, err
			}
			transList = append(transList, transPfd)
		}
	}
	return rsp, transList, err
}

//Creates a new subscription
func (af *afData) afAddPFDTransaction(nefCtx *nefContext,
	trans PfdManagement) (loc string, rsp nefSBRspData, err error) {

	nef := &nefCtx.nef
	/*Check if max subscription reached */
	if len(af.pfdtrans) >= nefCtx.cfg.MaxSubSupport {

		rsp.errorCode = 400
		rsp.pd.Title = "MAX Transaction Reached"
		return "", rsp, errors.New("MAX TRANS Created")
	}
	//Generate a unique transaction ID string
	transIDStr := strconv.Itoa(af.transIDnum)
	af.transIDnum++

	// TBD Validate for Duplicate Application ID

	var appIds []string
	var exist, create bool
	for key := range trans.PfdDatas {
		if nef.nefCheckPfdAppIDExists(key) {
			appIds = append(appIds, key)
			exist = true
		} else {
			create = true
		}
	}
	// if exist and !create send 500
	//else send 200 with pfdreport but without those Pfds.
	if exist {

		log.Infoln("Duplicate App Ids", appIds)

		pfdReport := generatePfdReport(appIds, "APP_ID_DUPLICATED")

		trans.PfdReports["APP_ID_DUPLICATED"] = pfdReport

		if !create {
			//remove everything from trans except pfd report as all
			//appIds have failed
			rsp.errorCode = 500
			rsp.pd.Title = "PFD applications provisioning unsuccessful"
			for key := range trans.PfdDatas {
				delete(trans.PfdDatas, key)
			}
			return "", rsp, errors.New("PFD unsuccessful")
		}

		for key := range appIds {
			delete(trans.PfdDatas, appIds[key])
		}
		// trans with pfd report added and remove Pfds with appIDs.

	}

	//Create PFD transaction data
	aftrans := afPfdTransaction{transID: transIDStr, pfdManagement: trans}

	/*rsp, err = nefSBUDRPost(&afsub, nefCtx, ti)

	if err != nil {

		//Return error
		return "", rsp, err
	}

	//Store Notification Destination URI
	afsub.afNotificationDestination = ti.NotificationDestination

	afsub.NEFSBGet = nefSBUDRGet
	afsub.NEFSBPut = nefSBUDRPut
	afsub.NEFSBPatch = nefSBUDRPatch
	afsub.NEFSBDelete = nefSBUDRDelete
	*/
	//Link the subscription with the AF
	af.pfdtrans[transIDStr] = &aftrans

	//Create Location URI
	loc = nefCtx.nef.locationURLPrefixPfd + af.afID + "/transactions/" +
		transIDStr

	af.pfdtrans[transIDStr].pfdManagement.Self = Link(loc)

	//Also update the self link in each application
	for k, v := range af.pfdtrans[transIDStr].pfdManagement.PfdDatas {

		/*Assign the application ID in the link */
		v.Self = Link(loc) + "/applications/" + Link(k)
		log.Infof("Application ID is %s", k)
		af.pfdtrans[transIDStr].pfdManagement.PfdDatas[k] = v

	}

	log.Infoln(" NEW AF PFD transaction added " + transIDStr)

	return loc, rsp, nil
}

// Generate the notification uri for PFD
func getNefLocationURLPrefixPfd(cfg *Config) string {

	var uri string
	// If http2 port is configured use it else http port
	if cfg.HTTP2Config.Endpoint != "" {
		uri = "https://" + cfg.NefAPIRoot +
			cfg.HTTP2Config.Endpoint
	} else {
		uri = "http://" + cfg.NefAPIRoot +
			cfg.HTTPConfig.Endpoint
	}
	uri += cfg.LocationPrefixPfd
	return uri

}
