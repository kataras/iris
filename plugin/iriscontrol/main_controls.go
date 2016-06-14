package iriscontrol

// for the main server
func (i *irisControlPlugin) StartServer() {
	if i.station.HTTPServer.IsListening() == false {
		if i.station.HTTPServer.IsSecure() {
			//listen with ListenTLS
			i.station.ListenTLS(i.station.Config.Server.ListeningAddr, i.station.Config.Server.CertFile, i.station.Config.Server.KeyFile)
		} else {
			//listen normal
			i.station.Listen(i.station.Config.Server.ListeningAddr)
		}
	}
}

func (i *irisControlPlugin) StopServer() {
	if i.station.HTTPServer.IsListening() {
		i.station.Close()
	}
}
