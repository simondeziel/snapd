// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2016 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

// Package snapstate implements the manager and state aspects responsible for the installation and removal of snaps.
package snapstate

import (
	"fmt"

	"gopkg.in/tomb.v2"

	"github.com/ubuntu-core/snappy/overlord/state"
	"github.com/ubuntu-core/snappy/progress"
	"github.com/ubuntu-core/snappy/snappy"
)

// SnapManager is responsible for the installation and removal of snaps.
type SnapManager struct {
	state   *state.State
	backend backendIF

	runner *state.TaskRunner
}

type installState struct {
	Name    string              `json:"name"`
	Channel string              `json:"channel"`
	Flags   snappy.InstallFlags `json:"flags,omitempty"`
}

type removeState struct {
	Name  string             `json:"name"`
	Flags snappy.RemoveFlags `json:"flags,omitempty"`
}

type purgeState struct {
	Name  string            `json:"name"`
	Flags snappy.PurgeFlags `json:"flags,omitempty"`
}

type rollbackState struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

type setActiveState struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// Manager returns a new snap manager.
func Manager(s *state.State) (*SnapManager, error) {
	runner := state.NewTaskRunner(s)
	backend := &defaultBackend{}
	m := &SnapManager{
		state:   s,
		backend: backend,
		runner:  runner,
	}

	runner.AddHandler("install-snap", m.doInstallSnap)
	runner.AddHandler("update-snap", m.doUpdateSnap)
	runner.AddHandler("remove-snap", m.doRemoveSnap)
	runner.AddHandler("purge-snap", m.doPurgeSnap)
	runner.AddHandler("rollback-snap", m.doRollbackSnap)
	runner.AddHandler("set-active-snap", m.doSetActiveSnap)

	// test handlers
	runner.AddHandler("fake-install-snap", func(t *state.Task, _ *tomb.Tomb) error {
		return nil
	})
	runner.AddHandler("fake-install-snap-error", func(t *state.Task, _ *tomb.Tomb) error {
		return fmt.Errorf("fake-install-snap-error errored")
	})

	return m, nil
}

func (m *SnapManager) doInstallSnap(t *state.Task, _ *tomb.Tomb) error {
	var inst installState
	t.State().Lock()
	if err := t.Get("state", &inst); err != nil {
		return err
	}
	t.State().Unlock()

	_, err := m.backend.Install(inst.Name, inst.Channel, inst.Flags, &progress.NullProgress{})
	return err
}

func (m *SnapManager) doUpdateSnap(t *state.Task, _ *tomb.Tomb) error {
	var inst installState
	t.State().Lock()
	if err := t.Get("state", &inst); err != nil {
		return err
	}
	t.State().Unlock()

	err := m.backend.Update(inst.Name, inst.Channel, inst.Flags, &progress.NullProgress{})
	return err
}

func (m *SnapManager) doRemoveSnap(t *state.Task, _ *tomb.Tomb) error {
	var rm removeState

	t.State().Lock()
	if err := t.Get("state", &rm); err != nil {
		return err
	}
	t.State().Unlock()

	name, _ := snappy.SplitDeveloper(rm.Name)
	err := m.backend.Remove(name, rm.Flags, &progress.NullProgress{})
	return err
}

func (m *SnapManager) doPurgeSnap(t *state.Task, _ *tomb.Tomb) error {
	var purge purgeState

	t.State().Lock()
	if err := t.Get("state", &purge); err != nil {
		return err
	}
	t.State().Unlock()

	name, _ := snappy.SplitDeveloper(purge.Name)
	err := m.backend.Purge(name, purge.Flags, &progress.NullProgress{})
	return err
}

func (m *SnapManager) doRollbackSnap(t *state.Task, _ *tomb.Tomb) error {
	var rollback rollbackState

	t.State().Lock()
	if err := t.Get("state", &rollback); err != nil {
		return err
	}
	t.State().Unlock()

	name, _ := snappy.SplitDeveloper(rollback.Name)
	_, err := m.backend.Rollback(name, rollback.Version, &progress.NullProgress{})
	return err
}

func (m *SnapManager) doSetActiveSnap(t *state.Task, _ *tomb.Tomb) error {
	var setActive setActiveState

	t.State().Lock()
	if err := t.Get("state", &setActive); err != nil {
		return err
	}
	t.State().Unlock()

	name, _ := snappy.SplitDeveloper(setActive.Name)
	return m.backend.SetActive(name, setActive.Active, &progress.NullProgress{})
}

// Ensure implements StateManager.Ensure.
func (m *SnapManager) Ensure() error {
	m.runner.Ensure()
	return nil
}

// Wait implements StateManager.Wait.
func (m *SnapManager) Wait() {
	m.runner.Wait()
}

// Stop implements StateManager.Stop.
func (m *SnapManager) Stop() {
	m.runner.Stop()
}
