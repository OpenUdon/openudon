package udonreport

import public "github.com/OpenUdon/openudon/udonreport"

type InventoryV5 = public.InventoryV5
type ObservationV5 = public.ObservationV5

func UnknownV5(i InventoryV5, state string) ObservationV5 { return public.UnknownV5(i, state) }
func ObserveV5(i InventoryV5, data []byte) ObservationV5  { return public.ObserveV5(i, data) }
