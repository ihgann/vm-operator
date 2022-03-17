// Copyright (c) 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package vcenter

import (
	goctx "context"
	"fmt"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/types"
)

// GetFolderByMoID returns the vim Folder for the MoID.
func GetFolderByMoID(
	ctx goctx.Context,
	finder *find.Finder,
	folderMoID string) (*object.Folder, error) {

	o, err := finder.ObjectReference(ctx, types.ManagedObjectReference{Type: "Folder", Value: folderMoID})
	if err != nil {
		return nil, err
	}

	return o.(*object.Folder), nil
}

// DoesChildFolderExist returns if the named child Folder exists under the parent Folder.
func DoesChildFolderExist(
	ctx goctx.Context,
	vimClient *vim25.Client,
	parentFolderMoID, childName string) (bool, error) {

	parentFolder := object.NewFolder(vimClient,
		types.ManagedObjectReference{Type: "Folder", Value: parentFolderMoID})

	childFolder, err := findChildFolder(ctx, parentFolder, childName)
	if err != nil {
		return false, err
	}

	return childFolder != nil, nil
}

// CreateFolder creates the named child Folder under the parent Folder.
func CreateFolder(
	ctx goctx.Context,
	vimClient *vim25.Client,
	parentFolderMoID, childName string) (string, error) {

	parentFolder := object.NewFolder(vimClient,
		types.ManagedObjectReference{Type: "Folder", Value: parentFolderMoID})

	childFolder, err := findChildFolder(ctx, parentFolder, childName)
	if err != nil {
		return "", err
	}

	if childFolder == nil {
		folder, err := parentFolder.CreateFolder(ctx, childName)
		if err != nil {
			return "", err
		}

		childFolder = folder
	}

	return childFolder.Reference().Value, nil
}

// DeleteChildFolder deletes the child Folder under the parent Folder.
func DeleteChildFolder(
	ctx goctx.Context,
	vimClient *vim25.Client,
	parentFolderMoID, childName string) error {

	parentFolder := object.NewFolder(vimClient,
		types.ManagedObjectReference{Type: "Folder", Value: parentFolderMoID})

	childFolder, err := findChildFolder(ctx, parentFolder, childName)
	if err != nil || childFolder == nil {
		return err
	}

	task, err := childFolder.Destroy(ctx)
	if err != nil {
		return err
	}

	if taskResult, err := task.WaitForResult(ctx); err != nil {
		if taskResult == nil || taskResult.Error == nil {
			return err
		}
		return fmt.Errorf("destroy Folder %s task failed: %w: %s",
			childFolder.Reference().Value, err, taskResult.Error.LocalizedMessage)
	}

	return nil
}

func findChildFolder(
	ctx goctx.Context,
	parentFolder *object.Folder,
	childName string) (*object.Folder, error) {

	objRef, err := object.NewSearchIndex(parentFolder.Client()).FindChild(ctx, parentFolder, childName)
	if err != nil {
		return nil, err
	} else if objRef == nil {
		return nil, nil
	}

	folder, ok := objRef.(*object.Folder)
	if !ok {
		return nil, fmt.Errorf("Folder child %q is not Folder but a %T", childName, objRef) //nolint
	}

	return folder, nil
}
