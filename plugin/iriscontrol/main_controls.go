package iriscontrol

// for the main server
func (i *irisControlPlugin) StartServer() {
	if i.station.Server().IsListening() == false {
		if i.station.Server().IsSecure() {
			//listen with ListenTLS
			i.station.ListenTLS(i.station.Server().Config.ListeningAddr, i.station.Server().Config.CertFile, i.station.Server().Config.KeyFile)
		} else {
			//listen normal
			i.station.Listen(i.station.Server().Config.ListeningAddr)
		}
	}
}

func (i *irisControlPlugin) StopServer() {
	if i.station.Server().IsListening() {
		i.station.Close()
	}
}
