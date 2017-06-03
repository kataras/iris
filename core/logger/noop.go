// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logger

var NoOpLogger = writerFunc(func([]byte) (int, error) { return -1, nil })
