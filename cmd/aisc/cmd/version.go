// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"github.com/aisc"

	"github.com/spf13/cobra"
)

func (c *command) initVersionCmd() {
	v := &cobra.Command{
		Use:   "version",
		Short: "Print version number",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(aisc.Version)
		},
	}
	v.SetOut(c.root.OutOrStdout())
	c.root.AddCommand(v)
}
