import { useContext } from "react";
import AuthContext from "../context/AuthProvier";

const useAuth = () => {
    return useContext(AuthContext);
}

export default useAuth;