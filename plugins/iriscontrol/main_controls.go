package iriscontrol

// for the main server
func (i *irisControlPlugin) StartServer() {
	if i.station.Server.IsRunning == false {
		if i.station.Server.IsSecure {
			//listen with ListenTLS
			i.station.ListenTLS(i.station.Server.ListeningAddr, i.station.Server.CertFile, i.station.Server.KeyFile)
		} else {
			//listen normal
			i.station.Listen(i.station.Server.ListeningAddr)
		}
	}
}

func (i *irisControlPlugin) StopServer() {
	if i.station.Server.IsRunning {
		i.station.Close()
	}
}
