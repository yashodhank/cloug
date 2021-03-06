package api

import "bytes"
import "crypto/sha512"
import "crypto/hmac"
import "crypto/rand"
import "encoding/json"
import "encoding/hex"
import "errors"
import "fmt"
import "io/ioutil"
import "net/url"
import "net/http"
import "strconv"
import "strings"
import "time"

type API struct {
	ApiId         string
	ApiKey        string
	ApiPartialKey string
}

func MakeAPI(id string, key string) (*API, error) {
	if len(id) != 16 {
		return nil, fmt.Errorf("API ID length must be 16, but parameter has length %d", len(id))
	} else if len(key) != 128 {
		return nil, fmt.Errorf("API key length must be 128, but parameter has length %d", len(key))
	}

	api := new(API)
	api.ApiId = id
	api.ApiKey = key
	api.ApiPartialKey = key[:64]
	return api, nil
}

func (api *API) request(category string, action string, params map[string]string, target interface{}) error {
	// construct URL
	targetUrl := LNDYNAMIC_API_URL
	targetUrl = strings.Replace(targetUrl, "{CATEGORY}", category, -1)
	targetUrl = strings.Replace(targetUrl, "{ACTION}", action, -1)

	// get raw parameters string
	if params == nil {
		params = make(map[string]string)
	}
	params["api_id"] = api.ApiId
	params["api_partialkey"] = api.ApiPartialKey
	rawParams, err := json.Marshal(params)
	if err != nil {
		return err
	}

	// HMAC signature with nonce
	nonce := fmt.Sprintf("%d", time.Now().Unix())
	handler := fmt.Sprintf("%s/%s/", category, action)
	hashTarget := fmt.Sprintf("%s|%s|%s", handler, string(rawParams), nonce)
	hasher := hmac.New(sha512.New, []byte(api.ApiKey))
	_, err = hasher.Write([]byte(hashTarget))
	if err != nil {
		return err
	}
	signature := hex.EncodeToString(hasher.Sum(nil))

	// perform request
	values := url.Values{}
	values.Set("handler", handler)
	values.Set("req", string(rawParams))
	values.Set("signature", signature)
	values.Set("nonce", nonce)
	byteBuffer := new(bytes.Buffer)
	byteBuffer.Write([]byte(values.Encode()))
	response, err := http.Post(targetUrl, "application/x-www-form-urlencoded", byteBuffer)
	if err != nil {
		return err
	}
	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	response.Body.Close()

	// decode JSON
	// we first decode into generic response for error checking; then into specific response to return
	var genericResponse GenericResponse
	err = json.Unmarshal(responseBytes, &genericResponse)

	if err != nil {
		return err
	} else if genericResponse.Success != "yes" {
		if genericResponse.Error != "" {
			return errors.New(genericResponse.Error)
		} else {
			return errors.New("backend call failed for unknown reason")
		}
	}

	if target != nil {
		err = json.Unmarshal(responseBytes, target)
		if err != nil {
			return err
		}
	}

	return nil
}

func (api *API) uid() string {
	bytes := make([]byte, 12)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

// virtual machines

func (api *API) VmCreateVolume(region string, hostname string, planIdentification int, volumeIdentification int) (int, error) {
	params := make(map[string]string)
	params["hostname"] = hostname
	params["region"] = region
	params["plan_id"] = fmt.Sprintf("%d", planIdentification)
	params["volume_id"] = fmt.Sprintf("%d", volumeIdentification)
	var response VmCreateResponse
	err := api.request("vm", "create", params, &response)
	if err != nil {
		return 0, err
	} else {
		id, err := strconv.Atoi(response.ID)
		if err != nil {
			return 0, err
		} else {
			return id, nil
		}
	}
}
func (api *API) VmCreateImage(region string, hostname string, planIdentification int, imageIdentification int) (int, error) {
	params := make(map[string]string)
	params["hostname"] = hostname
	params["region"] = region
	params["plan_id"] = fmt.Sprintf("%d", planIdentification)
	params["image_id"] = fmt.Sprintf("%d", imageIdentification)
	var response VmCreateResponse
	err := api.request("vm", "create", params, &response)
	if err != nil {
		return 0, err
	} else {
		id, err := strconv.Atoi(response.ID)
		if err != nil {
			return 0, err
		} else {
			return id, nil
		}
	}
}

func (api *API) vmAction(vmIdentification int, action string, params map[string]string) error {
	if params == nil {
		params = make(map[string]string)
	}
	params["vm_id"] = fmt.Sprintf("%d", vmIdentification)
	return api.request("vm", action, params, nil)
}

func (api *API) VmStart(vmIdentification int) error {
	return api.vmAction(vmIdentification, "start", nil)
}

func (api *API) VmStop(vmIdentification int) error {
	return api.vmAction(vmIdentification, "stop", nil)
}

func (api *API) VmReboot(vmIdentification int) error {
	return api.vmAction(vmIdentification, "reboot", nil)
}

func (api *API) VmDelete(vmIdentification int) error {
	return api.vmAction(vmIdentification, "delete", nil)
}

func (api *API) VmDiskSwap(vmIdentification int) error {
	return api.vmAction(vmIdentification, "diskswap", nil)
}

func (api *API) VmReimage(vmIdentification int, imageIdentification int) error {
	params := make(map[string]string)
	params["image_id"] = fmt.Sprintf("%d", imageIdentification)
	return api.vmAction(vmIdentification, "reimage", params)
}

func (api *API) VmVnc(vmIdentification int) (string, error) {
	params := make(map[string]string)
	params["vm_id"] = fmt.Sprintf("%d", vmIdentification)
	var response VmVncResponse
	err := api.request("vm", "vnc", params, &response)
	if err != nil {
		return "", err
	} else {
		return response.VncUrl, nil
	}
}

func (api *API) VmList() ([]VmStruct, error) {
	var response VmListResponse
	err := api.request("vm", "list", nil, &response)
	if err != nil {
		return nil, err
	} else {
		return response.Vms, nil
	}
}

func (api *API) VmInfo(vmIdentification int) (*VmStruct, *VmInfoStruct, error) {
	params := make(map[string]string)
	params["vm_id"] = fmt.Sprintf("%d", vmIdentification)
	var response VmInfoResponse
	err := api.request("vm", "info", params, &response)
	if err != nil {
		return nil, nil, err
	} else {
		return response.Extra, response.Info, nil
	}
}

func (api *API) VmSnapshot(vmIdentification int) (int, error) {
	// create snapshot with random label
	imageLabel := api.uid()
	params := make(map[string]string)
	params["vm_id"] = fmt.Sprintf("%d", vmIdentification)
	params["name"] = imageLabel
	var response ImageCreateResponse
	err := api.request("vm", "snapshot", params, &response)
	if err != nil {
		return 0, err
	} else {
		id, _ := strconv.Atoi(response.ID)
		return id, nil
	}
}

// images

func (api *API) ImageFetch(region string, location string, format string, virtio bool) (int, error) {
	// create an image with random label
	imageLabel := api.uid()
	params := make(map[string]string)
	params["region"] = region
	params["name"] = imageLabel
	params["location"] = location
	params["format"] = format
	if virtio {
		params["virtio"] = "yes"
	}
	var response ImageCreateResponse
	err := api.request("image", "fetch", params, &response)
	if err != nil {
		return 0, err
	} else {
		id, _ := strconv.Atoi(response.ID)
		return id, nil
	}
}

func (api *API) ImageDetails(imageIdentification int) (*Image, error) {
	params := make(map[string]string)
	params["image_id"] = fmt.Sprintf("%d", imageIdentification)
	var response ImageDetailsResponse
	err := api.request("image", "details", params, &response)
	if err != nil {
		return nil, err
	} else {
		return response.Image, nil
	}
}

func (api *API) ImageDelete(imageIdentification int) error {
	params := make(map[string]string)
	params["image_id"] = fmt.Sprintf("%d", imageIdentification)
	return api.request("image", "delete", params, nil)
}

func (api *API) ImageList(region string) ([]*Image, error) {
	params := make(map[string]string)
	if region != "" {
		params["region"] = region
	}
	var listResponse ImageListResponse
	err := api.request("image", "list", params, &listResponse)
	if err != nil {
		return nil, err
	} else {
		return listResponse.Images, nil
	}
}

// volumes

// Create a volume with the given size in gigabytes and image identification.
// If timeout is greater than zero, we will wait for the volume to become ready, or return error if timeout is exceeded.
// Otherwise, we return immediately without error.
func (api *API) VolumeCreate(region string, size int, imageIdentification int, timeout time.Duration) (int, error) {
	// create a volume with random label
	volumeLabel := api.uid()
	params := make(map[string]string)
	params["region"] = region
	params["label"] = volumeLabel
	params["size"] = fmt.Sprintf("%d", size)
	params["image"] = fmt.Sprintf("%d", imageIdentification)
	var response VolumeCreateResponse
	err := api.request("volume", "create", params, &response)
	if err != nil {
		return 0, err
	} else {
		id, _ := strconv.Atoi(response.ID)
		return id, nil
	}
}

func (api *API) VolumeDelete(region string, volumeIdentification int) error {
	params := make(map[string]string)
	params["region"] = region
	params["volume_id"] = fmt.Sprintf("%d", volumeIdentification)
	return api.request("volume", "delete", params, nil)
}

// plans

func (api *API) PlanList() ([]*Plan, error) {
	var listResponse PlanListResponse
	err := api.request("plan", "list", nil, &listResponse)
	return listResponse.Plans, err
}
