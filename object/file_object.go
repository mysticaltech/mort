package object

import (
	"errors"
	"strings"
	"path"

	"mort/log"
	"mort/config"
	"mort/transforms"
)


func presetToTransform(preset config.PresetsYaml) transforms.Transforms {
	var trans transforms.Transforms
	filters := preset.Filters

	if len(filters.Thumbnail.Size) > 0 {
		trans.ResizeT(filters.Thumbnail.Size, filters.Thumbnail.Mode == "outbound")
	}

	if len(filters.SmartCrop.Size) > 0 {
		trans.CropT(filters.SmartCrop.Size, filters.SmartCrop.Mode == "outbound")
	}

	if len(filters.Crop.Size) > 0 {
		trans.CropT(filters.Crop.Size, filters.Crop.Mode == "outbound")
	}

	trans.Quality = preset.Quality

	return trans
}

type FileObject struct {
	Uri        string                `json:"uri"`
	Bucket     string                `json:"bucket"`
	Key        string                `json:"key"`
	Transforms transforms.Transforms `json:"transforms"`
	Storage    config.Storage        `json:"storage"`
	Parent     *FileObject
}

func NewFileObject(uri string, mortConfig *config.Config) (*FileObject, error) {
	obj := FileObject{}
	obj.Uri = uri

	err := obj.decode(mortConfig)
	log.Log().Infow("New FileObject", "path", uri,  "key", obj.Key, "bucket", obj.Bucket, "storage", obj.Storage,
		"hasTransforms", obj.HasTransform(), "hasParent" , obj.HasParent())
	return &obj, err
}

func (self *FileObject) decode(mortConfig *config.Config) error {
	elements := strings.Split(self.Uri, "/")

	self.Bucket = elements[1]
	if len(elements) > 2 {
		self.Key = "/" + strings.Join(elements[2:], "/")
	}

	if bucket, ok := mortConfig.Buckets[self.Bucket]; ok {
		err := self.decodeKey(bucket, mortConfig)
		if self.HasTransform() {
			self.Storage = bucket.Storages.Transform()
		} else {
			self.Storage = bucket.Storages.Basic()
		}
		return err

	} else {
		return errors.New("Unknown bucket")
	}

	return nil
}

func (self *FileObject) decodeKey(bucket config.Bucket, mortConfig *config.Config) error {
	if bucket.Transform == nil {
		return nil
	}

	trans := bucket.Transform
	matches := trans.PathRegexp.FindStringSubmatch(self.Key)
	if len(matches) < 3 {
		return nil
	}

	presetName := string(matches[trans.Order.PresetName+1])
	parent := "/" + string(matches[trans.Order.Parent+1])

	self.Transforms = presetToTransform(bucket.Transform.Presets[presetName])
	if bucket.Transform.ParentBucket != "" {
		parent = "/" + path.Join(bucket.Transform.ParentBucket, parent)
	}
	parentObj, err := NewFileObject(parent, mortConfig)
	parentObj.Storage = bucket.Storages.Get(bucket.Transform.ParentStorage)
	self.Parent = parentObj
	return err
}

func (self *FileObject) HasParent() bool {
	return self.Parent != nil
}

func (self *FileObject) HasTransform() bool {
	return self.Transforms.NotEmpty == true
}
