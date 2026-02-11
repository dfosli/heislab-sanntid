package requests

import (
	"heislab-sanntid/elevator/elevio"
	"heislab-sanntid/elevator/elev_struct"
	"heislab-sanntid/config"
)

const (
	N_FLOORS  int = config.N_FLOORS
	N_BUTTONS int = config.N_BUTTONS
)



""" c kode
static int requests_above(Elevator e){
    for(int f = e.floor+1; f < N_FLOORS; f++){
        for(int btn = 0; btn < N_BUTTONS; btn++){
            if(e.requests[f][btn]){
                return 1;
            }
        }
    }
    return 0;
}

static int requests_below(Elevator e){
    for(int f = 0; f < e.floor; f++){
        for(int btn = 0; btn < N_BUTTONS; btn++){
            if(e.requests[f][btn]){
                return 1;
            }
        }
    }
    return 0;
}

static int requests_here(Elevator e){
    for(int btn = 0; btn < N_BUTTONS; btn++){
        if(e.requests[e.floor][btn]){
            return 1;
        }
    }
    return 0;
}
"""
