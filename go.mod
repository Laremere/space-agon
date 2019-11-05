module github.com/googleforgames/space-agon

// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

go 1.13

require (
	agones.dev/agones v1.1.0
	golang.org/x/net v0.0.0-20190926025831-c00fd9afed17
	google.golang.org/grpc v1.21.1
	k8s.io/apimachinery v0.0.0-20190221084156-01f179d85dbc
	k8s.io/client-go v9.0.0+incompatible
	open-match.dev/open-match v0.8.0-rc.1
)
