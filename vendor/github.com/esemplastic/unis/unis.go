// Copyright 2017 Γεράσιμος Μαρόπουλος. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unis

import (
	"github.com/esemplastic/unis/logger"
)

// Logger prints only errors that should "panic" (I don't like to call panic inside packages).
//
// The user can assign this variable and change it in order to meet his project's
// meetings, Logger is just a func which accepts a string (func(string)).
//
// To disable the logger assign the unis.Logger to a new logger.NewProd() from "/logger" package.
var Logger = logger.NewDev()
