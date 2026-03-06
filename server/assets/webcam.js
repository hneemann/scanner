var webcam = {
    // (A) INITIALIZE - GET USER PERMISSION TO ACCESS CAMERA
    vFlip: false,
    hVid: null,
    uploadDiv:null,
    successDiv:null,
    failedDiv:null,
    init: () => {

        webcam.uploadDiv=document.getElementById("upload");
        webcam.successDiv=document.getElementById("success");
        webcam.successDocDiv=document.getElementById("successDoc");
        webcam.failedDiv=document.getElementById("failed");

        navigator.mediaDevices.getUserMedia({audio: false, video: {width: { ideal: 4096 }, facingMode: { exact: 'environment' }}})
            .then(stream => {
                // (A1) GET HTML ELEMENTS
                webcam.hVid = document.getElementById("cam-live");

                // (A2) "LIVE FEED" WEB CAM TO <VIDEO>
                webcam.hVid.srcObject = stream;


                // (A3) ENABLE BUTTONS
                //document.getElementById("cam-take").disabled = false;
                document.getElementById("cam-flip").disabled = false;
                document.getElementById("cam-upload").disabled = false;
            })
            .catch(err => console.error(err));

    },

    // (B) SNAP VIDEO FRAME TO CANVAS
    snap: () => {
        // (B1) CREATE NEW CANVAS
        let cv = document.createElement("canvas"),
            cx = cv.getContext("2d");

        // (B2) CAPTURE VIDEO FRAME TO CANVAS
        cv.width = webcam.hVid.videoWidth;
        cv.height = webcam.hVid.videoHeight;
        cx.drawImage(webcam.hVid, 0, 0, webcam.hVid.videoWidth, webcam.hVid.videoHeight);

        // (B3) DONE
        return cv;
    },

    flip: () => {
        webcam.vFlip = !webcam.vFlip;
        if (webcam.vFlip) {
            webcam.hVid.style.transform = "matrix(-1,0,0,-1,0,0)";
        } else {
            webcam.hVid.style.transform = "";
        }
    },

    // (E) UPLOAD SNAPSHOT TO SERVER
    upload: () => {
        webcam.uploadDiv.style.visibility = "visible";

        // (E1) APPEND SCREENSHOT TO DATA OBJECT
        webcam.snap().toBlob(blob=>{
            if (blob === null) {
                webcam.uploadDiv.style.visibility = "hidden";
                webcam.failedDiv.style.visibility = "visible";
                return;
            }
            let formData = new FormData();
            formData.append("scan", blob);
            formData.append("flip", webcam.vFlip?"true":"false");
            // (E2) UPLOAD SCREENSHOT TO SERVER
            fetch("/store/", {body: formData, method: "post", signal: AbortSignal.timeout(30000)})
                .then(res => {
                    if (res.status !== 200) {
                        document.getElementById("failedStatus").innerHTML="(response: "+res.status+")";
                        webcam.uploadDiv.style.visibility = "hidden";
                        webcam.failedDiv.style.visibility = "visible";
                    }
                    return res.text()}
                )
                .catch(reason => {
                    document.getElementById("failedCause").innerHTML=reason.toString();
                    webcam.uploadDiv.style.visibility = "hidden";
                    webcam.failedDiv.style.visibility = "visible";
                })
                .then(txt => {
                   document.getElementById("failedCause").innerHTML=txt.toString();
                   webcam.uploadDiv.style.visibility = "hidden";
                   if (webcam.failedDiv.style.visibility!=="visible") {
                       webcam.successDiv.style.visibility = "visible";
                   }
                });
        },"image/jpeg", 0.9);
    }
};
window.addEventListener("load", webcam.init);

function hide(id) {
    let el=document.getElementById(id);
    if (el) {
        el.style.visibility = "hidden";
    }
}

function createPDF() {
    hide('success');
    webcam.uploadDiv.style.visibility = "visible";

    fetch("/create/", {method: "get", signal: AbortSignal.timeout(30000)})
        .then(res => {
            if (res.status !== 200) {
                document.getElementById("failedStatus").innerHTML="(status: "+res.status+")";
                webcam.uploadDiv.style.visibility = "hidden";
                webcam.failedDiv.style.visibility = "visible";
            }
            return res.text()
        })
        .catch(reason => {
            document.getElementById("failedCause").innerHTML=reason.toString();
            webcam.uploadDiv.style.visibility = "hidden";
            webcam.failedDiv.style.visibility = "visible";
        })
        .then(txt => {
            document.getElementById("failedCause").innerHTML=txt.toString();
            webcam.uploadDiv.style.visibility = "hidden";
            if (webcam.failedDiv.style.visibility!=="visible") {
                webcam.successDocDiv.style.visibility = "visible";
            }
        });

}